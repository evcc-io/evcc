package charger

import (
	"context"
	"errors"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/eebus-go/usecases/ma/mdt"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

// EEBusOHPCF controls a remote heat pump compressor via the EEBus OHPCF use case
// (Optimization of Self-Consumption by Heat Pump Compressor Flexibility).
//
// The compressor announces an optional power consumption that the CEM may
// schedule, pause, resume or abort. evcc models this as an on/off switch:
// enabling the charger schedules or resumes the optional consumption, disabling
// it pauses or aborts the running process.
type EEBusOHPCF struct {
	*embed
	cem *eebus.CustomerEnergyManagement
	ma  *eebus.MonitoringAppliance
	eg  *eebus.EnergyGuard

	ctx     context.Context
	reboost time.Duration

	mu          sync.RWMutex
	log         *util.Logger
	compressor  spineapi.EntityRemoteInterface
	mpcEntity   spineapi.EntityRemoteInterface
	dhwEntity   spineapi.EntityRemoteInterface
	egLpcEntity spineapi.EntityRemoteInterface
	enabled     bool
	reboosting  bool

	connector *eebus.Connector
}

// errNotConnected is returned whenever the compressor entity is not (yet) available.
var errNotConnected = errors.New("not connected")

func init() {
	registry.AddCtx("eebus-ohpcf", NewEEBusOHPCFFromConfig)
}

// NewEEBusOHPCFFromConfig creates an EEBus OHPCF charger from generic config
func NewEEBusOHPCFFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed   `mapstructure:",squash"`
		Ski     string
		Ip      string
		Reboost time.Duration
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice, api.SwitchDevice},
		},
		Reboost: 10 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBusOHPCF(ctx, &cc.embed, cc.Ski, cc.Ip, cc.Reboost)
}

// NewEEBusOHPCF creates an EEBus OHPCF charger, registers it with the EEBus
// instance and waits for the connection.
func NewEEBusOHPCF(ctx context.Context, embed *embed, ski, ip string, reboost time.Duration) (api.Charger, error) {
	inst, err := eebus.Instance()
	if err != nil {
		return nil, err
	}

	c := &EEBusOHPCF{
		embed:     embed,
		log:       util.NewLogger("eebus-ohpcf"),
		cem:       inst.CustomerEnergyManagement(),
		ma:        inst.MonitoringAppliance(),
		eg:        inst.EnergyGuard(),
		connector: eebus.NewConnector(),
		ctx:       ctx,
		reboost:   reboost,
	}

	if err := inst.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.connector.Wait(ctx); err != nil {
		inst.UnregisterDevice(ski, c)
		return nil, err
	}

	// unregister device when context is cancelled (e.g. UI config validation)
	go func() {
		<-ctx.Done()
		inst.UnregisterDevice(ski, c)
	}()

	return c, nil
}

var _ eebus.Device = (*EEBusOHPCF)(nil)

// Connect implements the eebus.Device interface
func (c *EEBusOHPCF) Connect(connected bool) {
	c.connector.Connect(connected)

	if connected {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.compressor = nil
	c.mpcEntity = nil
	c.dhwEntity = nil
	c.egLpcEntity = nil
}

// UseCaseEvent implements the eebus.Device interface
func (c *EEBusOHPCF) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	// device/entity removal fires the use case update event with a nil entity
	if entity == nil {
		return
	}

	switch event {
	case ohpcf.DataUpdateConsumptionState:
		c.mu.Lock()
		c.compressor = entity
		c.mu.Unlock()

		// react immediately to a freshly announced schedule/resume opportunity
		// instead of waiting for the next reboost tick, which may miss it (#31549)
		if c.lastEnabled() {
			if err := c.apply(); err != nil {
				c.log.DEBUG.Printf("apply: %v", err)
			}
		}

	case ohpcf.UseCaseSupportUpdate,
		ohpcf.DataUpdateRequestedPowerEstimate,
		ohpcf.DataUpdateRequestedPowerMax,
		ohpcf.DataUpdateConsumptionIsStoppable,
		ohpcf.DataUpdateConsumptionIsPausable,
		ohpcf.DataUpdateConsumptionStartTime,
		ohpcf.DataUpdateMinimalRunDuration,
		ohpcf.DataUpdateMinimalPauseDuration:
		c.mu.Lock()
		c.compressor = entity
		c.mu.Unlock()

	// Monitoring Appliance MPC provides the measured power consumption
	case mpc.UseCaseSupportUpdate:
		c.mu.Lock()
		// use most specific selector
		if c.mpcEntity == nil || len(entity.Address().Entity) < len(c.mpcEntity.Address().Entity) {
			c.mpcEntity = entity
		}
		c.mu.Unlock()

	// Monitoring Appliance MDT provides the DHW temperature
	case mdt.UseCaseSupportUpdate, mdt.DataUpdateTemperature:
		c.mu.Lock()
		c.dhwEntity = entity
		c.mu.Unlock()

	// Energy Guard LPC carries the §14a/LPC consumption limit
	case lpc.UseCaseSupportUpdate:
		c.mu.Lock()
		// use most specific selector
		if c.egLpcEntity == nil || len(entity.Address().Entity) < len(c.egLpcEntity.Address().Entity) {
			c.egLpcEntity = entity
		}
		c.mu.Unlock()
	}
}

func (c *EEBusOHPCF) connectedCompressor() (spineapi.EntityRemoteInterface, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.compressor, c.compressor != nil
}

func (c *EEBusOHPCF) setEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled = enabled
}

func (c *EEBusOHPCF) lastEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.enabled
}

// ohpcfStatus maps the compressor process state to a charge status: running is
// consuming (C), any other connected state is standby (B). Disconnected (A) is handled in Status.
func ohpcfStatus(state ucapi.CompressorPowerConsumptionStateType) api.ChargeStatus {
	if state == ucapi.CompressorPowerConsumptionStateRunning {
		return api.StatusC
	}
	return api.StatusB
}

var _ api.Charger = (*EEBusOHPCF)(nil)

// Status implements the api.Charger interface
func (c *EEBusOHPCF) Status() (api.ChargeStatus, error) {
	entity, ok := c.connectedCompressor()
	if !ok {
		return api.StatusNone, errNotConnected
	}

	state, err := c.cem.OHPCF.PowerConsumptionProcessState(entity)
	if err != nil {
		// connected but no flexibility announced yet: standby, not disconnected
		return api.StatusB, nil
	}

	return ohpcfStatus(state), nil
}

// Enabled reports the commanded on/off intent; Status reflects the actual
// compressor state.
func (c *EEBusOHPCF) Enabled() (bool, error) {
	if _, ok := c.connectedCompressor(); !ok {
		return false, errNotConnected
	}

	return c.lastEnabled(), nil
}

// Enable schedules/resumes the optional consumption when on, pauses/aborts it
// when off; while on a reboost loop reschedules newly announced consumption.
func (c *EEBusOHPCF) Enable(enable bool) error {
	c.setEnabled(enable)

	if enable {
		c.startReboost()
	}

	return c.apply()
}

// startReboost launches the reboost loop, unless one is already running or no
// reboost interval is configured.
func (c *EEBusOHPCF) startReboost() {
	if c.reboost <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.reboosting {
		return
	}

	c.reboosting = true
	go c.reboostLoop()
}

// reboostLoop reschedules a freshly announced optional consumption after each
// reboost interval; it exits when the charger is disabled or the context ends.
func (c *EEBusOHPCF) reboostLoop() {
	defer func() {
		c.mu.Lock()
		c.reboosting = false
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(c.reboost):
			if !c.lastEnabled() {
				return
			}
			if err := c.apply(); err != nil {
				c.log.DEBUG.Printf("reboost: %v", err)
			}
		}
	}
}

type ohpcfAction int

const (
	ohpcfNone ohpcfAction = iota
	ohpcfSchedule
	ohpcfResume
	ohpcfStop
)

// ohpcfControlAction returns the command needed to reach the desired on/off
// state; it returns an action only on a state transition, so repeats are no-ops.
func ohpcfControlAction(state ucapi.CompressorPowerConsumptionStateType, enable bool) ohpcfAction {
	if enable {
		switch state {
		case ucapi.CompressorPowerConsumptionStateAvailable:
			return ohpcfSchedule
		case ucapi.CompressorPowerConsumptionStatePaused:
			return ohpcfResume
		}
		return ohpcfNone
	}

	switch state {
	case ucapi.CompressorPowerConsumptionStateRunning,
		ucapi.CompressorPowerConsumptionStateScheduled:
		return ohpcfStop
	}

	return ohpcfNone
}

// stop pauses the optional consumption if the compressor permits it, otherwise
// it aborts the process.
func (c *EEBusOHPCF) stop(entity spineapi.EntityRemoteInterface) error {
	if pausable, err := c.cem.OHPCF.ConsumptionIsPausable(entity); err == nil && pausable {
		return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
			return c.cem.OHPCF.PausePowerConsumptionProcess(entity, cb)
		})
	}

	if stoppable, err := c.cem.OHPCF.ConsumptionIsStoppable(entity); err == nil && stoppable {
		return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
			return c.cem.OHPCF.AbortPowerConsumptionProcess(entity, cb)
		})
	}

	return api.ErrNotAvailable
}

// MaxCurrent implements the api.Charger interface. OHPCF is on/off and cannot
// be modulated, so the offered current is ignored.
func (c *EEBusOHPCF) MaxCurrent(int64) error {
	return c.apply()
}

var _ api.Dimmer = (*EEBusOHPCF)(nil)

// Dimmed implements the api.Dimmer interface, reporting whether a §14a/LPC
// consumption limit is currently active on the heat pump.
func (c *EEBusOHPCF) Dimmed() (bool, error) {
	c.mu.RLock()
	entity := c.egLpcEntity
	c.mu.RUnlock()

	if entity == nil || !c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(entity, eebus.LPCLimit) {
		return false, api.ErrNotAvailable
	}

	limit, err := c.eg.EgLPCInterface.ConsumptionLimit(entity)
	if err != nil {
		// scenario announced but no usable value yet
		if errors.Is(err, eebusapi.ErrDataNotAvailable) ||
			errors.Is(err, eebusapi.ErrMetadataNotAvailable) ||
			errors.Is(err, eebusapi.ErrDataInvalid) {
			return false, api.ErrNotAvailable
		}
		return false, err
	}

	// an active limit means dimmed; the applied §14a limit value is 0W, so a
	// value-based check would never report the dimmed state and never release it
	return limit.IsActive, nil
}

// Dim implements the api.Dimmer interface. It writes a §14a/LPC consumption
// limit (fixed 0W safe limit) to the heat pump while dimmed, releasing it otherwise.
func (c *EEBusOHPCF) Dim(dim bool) error {
	c.mu.RLock()
	entity := c.egLpcEntity
	c.mu.RUnlock()

	if entity == nil || !c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(entity, eebus.LPCLimit) {
		return api.ErrNotAvailable
	}

	// TODO: change api.Dimmer to make the limit configurable; use a fixed 0W safe limit for now
	return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
		return c.eg.EgLPCInterface.WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: 0, IsActive: dim}, cb)
	})
}

// apply issues the command to align the optional consumption with the on/off
// intent. It is idempotent: ohpcfControlAction only acts on a state transition.
func (c *EEBusOHPCF) apply() error {
	entity, ok := c.connectedCompressor()
	if !ok {
		return errNotConnected
	}

	state, err := c.cem.OHPCF.PowerConsumptionProcessState(entity)
	if err != nil {
		// no process state announced yet, nothing to control
		return nil
	}

	switch ohpcfControlAction(state, c.lastEnabled()) {
	case ohpcfSchedule:
		return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
			// 0 = start immediately (relative schedule, see SchedulePowerConsumptionProcess)
			return c.cem.OHPCF.SchedulePowerConsumptionProcess(entity, 0, cb)
		})
	case ohpcfResume:
		return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
			return c.cem.OHPCF.ResumePowerConsumptionProcess(entity, cb)
		})
	case ohpcfStop:
		return c.stop(entity)
	}

	return nil
}

var _ api.PowerLimiter = (*EEBusOHPCF)(nil)

// GetMinMaxPower implements the api.PowerLimiter interface, reporting the
// optional consumption as expected min/max or ErrNotAvailable if none.
func (c *EEBusOHPCF) GetMinMaxPower() (float64, float64, error) {
	entity, ok := c.connectedCompressor()
	if !ok {
		return 0, 0, errNotConnected
	}

	if power, _ := c.cem.OHPCF.RequestedPowerEstimate(entity); power > 0 {
		return power, power, nil
	}

	if power, _ := c.cem.OHPCF.RequestedPowerMax(entity); power > 0 {
		return power, power, nil
	}

	return 0, 0, api.ErrNotAvailable
}

var _ api.Meter = (*EEBusOHPCF)(nil)

// CurrentPower implements the api.Meter interface and reports the heat pump's
// measured power consumption via the MPC use case.
func (c *EEBusOHPCF) CurrentPower() (float64, error) {
	c.mu.RLock()
	entity := c.mpcEntity
	c.mu.RUnlock()

	if entity == nil || !c.ma.MaMPCInterface.IsScenarioAvailableAtEntity(entity, eebus.MPCPower) {
		return 0, api.ErrNotAvailable
	}

	power, err := c.ma.MaMPCInterface.Power(entity)
	if err != nil {
		return 0, eebus.WrapError(err)
	}

	return power, nil
}

var _ api.Battery = (*EEBusOHPCF)(nil)

// Soc implements the api.Battery interface and reports the heat pump's domestic
// hot water temperature in °C via the MDT use case.
func (c *EEBusOHPCF) Soc() (float64, error) {
	c.mu.RLock()
	entity := c.dhwEntity
	c.mu.RUnlock()

	if entity == nil || !c.ma.MaMDTInterface.IsScenarioAvailableAtEntity(entity, eebus.MDTTemperature) {
		return 0, api.ErrNotAvailable
	}

	temp, err := c.ma.MaMDTInterface.Temperature(entity, model.UnitOfMeasurementTypedegC)
	if err != nil {
		return 0, eebus.WrapError(err)
	}

	return temp, nil
}
