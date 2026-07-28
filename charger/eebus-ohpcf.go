package charger

import (
	"context"
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

	mu         sync.RWMutex
	log        *util.Logger
	compressor spineapi.EntityRemoteInterface
	mpc        spineapi.EntityRemoteInterface
	dhw        spineapi.EntityRemoteInterface
	egLpc      spineapi.EntityRemoteInterface
	enabled    bool
	reboosting bool
	dimmed     bool // last limit written, re-stated on reconnect

	connector *eebus.Connector
}

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
	c.mpc = nil
	c.dhw = nil
	c.egLpc = nil
}

// UseCaseEvent implements the eebus.Device interface
func (c *EEBusOHPCF) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	// device removal fires the use case update event with a nil entity
	if entity == nil {
		return
	}

	switch event {
	case ohpcf.UseCaseSupportUpdate:
		c.mu.Lock()
		c.compressor = eebus.UpdateEntity(c.cem.OHPCF, c.compressor, entity)
		c.mu.Unlock()

	case ohpcf.DataUpdateConsumptionState:
		// react immediately to a freshly announced schedule/resume opportunity
		// instead of waiting for the next reboost tick, which may miss it (#31549)
		if c.lastEnabled() {
			if err := c.apply(true); err != nil {
				c.log.DEBUG.Printf("apply: %v", err)
			}
		}

	// Monitoring Appliance MPC provides the measured power consumption
	case mpc.UseCaseSupportUpdate:
		c.mu.Lock()
		c.mpc = eebus.UpdateEntity(c.ma.MaMPCInterface, c.mpc, entity)
		c.mu.Unlock()

	// Monitoring Appliance MDT provides the DHW temperature
	case mdt.UseCaseSupportUpdate:
		c.mu.Lock()
		c.dhw = eebus.UpdateEntity(c.ma.MaMDTInterface, c.dhw, entity)
		c.mu.Unlock()

	// Energy Guard LPC carries the §14a/LPC consumption limit
	case lpc.UseCaseSupportUpdate:
		c.mu.Lock()
		prev := c.egLpc
		c.egLpc = eebus.UpdateEntity(c.eg.EgLPCInterface, c.egLpc, entity)

		if c.egLpc != nil && c.egLpc != prev {
			// [LPC-913]: state the limit to the newly available CS
			go eebus.AssertLimit(c.ctx, c.log, func() error { return c.Dim(c.lastDimmed()) })
		}
		c.mu.Unlock()
	}
}

// compressorEntity returns the compressor entity or ErrNotConnected while the
// OHPCF use case is not (yet) available at it.
func (c *EEBusOHPCF) compressorEntity() (spineapi.EntityRemoteInterface, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.compressor == nil {
		return nil, eebus.ErrNotConnected
	}

	return eebus.RequiredEntity(c.cem.OHPCF, eebus.OHPCFMonitor, c.compressor)
}

// mpcEntity returns the entity providing the measured power consumption
func (c *EEBusOHPCF) mpcEntity() spineapi.EntityRemoteInterface {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.mpc
}

// dhwEntity returns the entity providing the domestic hot water temperature
func (c *EEBusOHPCF) dhwEntity() spineapi.EntityRemoteInterface {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.dhw
}

// egLpcEntity returns the entity carrying the §14a/LPC consumption limit
func (c *EEBusOHPCF) egLpcEntity() spineapi.EntityRemoteInterface {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.egLpc
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

func (c *EEBusOHPCF) lastDimmed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.dimmed
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
	entity, err := c.compressorEntity()
	if err != nil {
		return api.StatusNone, err
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
	if _, err := c.compressorEntity(); err != nil {
		return false, err
	}

	return c.lastEnabled(), nil
}

// Enable schedules/resumes the optional consumption when on, pauses/aborts it
// when off; while on a reboost loop reschedules newly announced consumption.
func (c *EEBusOHPCF) Enable(enable bool) error {
	// record the intent only once accepted, otherwise Enabled() would report a
	// state the compressor never reached and the loadpoint runs out of sync
	if err := c.apply(enable); err != nil {
		return err
	}

	c.setEnabled(enable)

	if enable {
		c.startReboost()
	}

	return nil
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
			if err := c.apply(true); err != nil {
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
	return c.apply(c.lastEnabled())
}

var _ api.Dimmer = (*EEBusOHPCF)(nil)

// Dimmed implements the api.Dimmer interface, reporting whether a §14a/LPC
// consumption limit is currently active on the heat pump.
func (c *EEBusOHPCF) Dimmed() (bool, error) {
	limit, err := eebus.ReadValue(c.eg.EgLPCInterface, eebus.LPCLimit, c.egLpcEntity(), c.eg.EgLPCInterface.ConsumptionLimit)
	if err != nil {
		return false, err
	}

	// an active limit means dimmed; the applied §14a limit value is 0W, so a
	// value-based check would never report the dimmed state and never release it
	return limit.IsActive, nil
}

// Dim implements the api.Dimmer interface. It writes a §14a/LPC consumption
// limit (fixed 0W safe limit) to the heat pump while dimmed, releasing it otherwise.
func (c *EEBusOHPCF) Dim(dim bool) error {
	entity := c.egLpcEntity()

	if entity == nil || !c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(entity, eebus.LPCLimit) {
		return api.ErrNotAvailable
	}

	// TODO: change api.Dimmer to make the limit configurable; use a fixed 0W safe limit for now
	if err := eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
		return c.eg.EgLPCInterface.WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: 0, IsActive: dim}, cb)
	}); err != nil {
		return err
	}

	c.mu.Lock()
	c.dimmed = dim
	c.mu.Unlock()

	return nil
}

// apply issues the command to align the optional consumption with the on/off
// intent. It is idempotent: ohpcfControlAction only acts on a state transition.
func (c *EEBusOHPCF) apply(enable bool) error {
	entity, err := c.compressorEntity()
	if err != nil {
		return err
	}

	state, err := c.cem.OHPCF.PowerConsumptionProcessState(entity)
	if err != nil {
		// no process state announced yet, nothing to control
		return nil
	}

	switch ohpcfControlAction(state, enable) {
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
	entity, err := c.compressorEntity()
	if err != nil {
		return 0, 0, err
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
	return eebus.ReadValue(c.ma.MaMPCInterface, eebus.MPCPower, c.mpcEntity(), c.ma.MaMPCInterface.Power)
}

var _ api.Battery = (*EEBusOHPCF)(nil)

// Soc implements the api.Battery interface and reports the heat pump's domestic
// hot water temperature in °C via the MDT use case.
func (c *EEBusOHPCF) Soc() (float64, error) {
	return eebus.ReadValue(c.ma.MaMDTInterface, eebus.MDTTemperature, c.dhwEntity(),
		func(entity spineapi.EntityRemoteInterface) (float64, error) {
			return c.ma.MaMDTInterface.Temperature(entity, model.UnitOfMeasurementTypedegC)
		})
}
