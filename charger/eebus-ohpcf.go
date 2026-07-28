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
	ctx     context.Context
	reboost time.Duration

	log   *util.Logger
	ohpcf *eebus.Entity[ucapi.CemOHPCFInterface]
	mpc   *eebus.Entity[ucapi.MaMPCInterface]
	mdt   *eebus.Entity[ucapi.MaMDTInterface]
	lpc   *eebus.Entity[ucapi.EgLPCInterface]

	mu         sync.RWMutex
	enabled    bool
	reboosting bool

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

	cem := inst.CustomerEnergyManagement()
	ma := inst.MonitoringAppliance()
	eg := inst.EnergyGuard()

	c := &EEBusOHPCF{
		embed:     embed,
		log:       util.NewLogger("eebus-ohpcf"),
		connector: eebus.NewConnector(),
		ctx:       ctx,
		reboost:   reboost,
		ohpcf:     eebus.NewEntity(cem.OHPCF),
		mpc:       eebus.NewEntity(ma.MaMPCInterface),
		mdt:       eebus.NewEntity(ma.MaMDTInterface),
		lpc:       eebus.NewEntity(eg.EgLPCInterface),
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

	c.ohpcf.Set(nil)
	c.mpc.Set(nil)
	c.mdt.Set(nil)
	c.lpc.Set(nil)
}

// UseCaseEvent implements the eebus.Device interface
func (c *EEBusOHPCF) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	// device removal fires the use case update event with a nil entity
	if entity == nil {
		return
	}

	switch event {
	case ohpcf.UseCaseSupportUpdate:
		c.ohpcf.Update(entity)

	case ohpcf.DataUpdateConsumptionState:
		// react immediately to a freshly announced schedule/resume opportunity
		// instead of waiting for the next reboost tick, which may miss it (#31549)
		if c.lastEnabled() {
			if err := c.apply(); err != nil {
				c.log.DEBUG.Printf("apply: %v", err)
			}
		}

	// Monitoring Appliance MPC provides the measured power consumption
	case mpc.UseCaseSupportUpdate:
		c.mpc.Update(entity)

	// Monitoring Appliance MDT provides the DHW temperature
	case mdt.UseCaseSupportUpdate:
		c.mdt.Update(entity)

	// Energy Guard LPC carries the §14a/LPC consumption limit
	case lpc.UseCaseSupportUpdate:
		c.lpc.Update(entity)
	}
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
	if _, err := c.ohpcf.Required(); err != nil {
		return api.StatusNone, err
	}

	state, err := c.ohpcf.Read(ucapi.CemOHPCFInterface.PowerConsumptionProcessState)
	if err != nil {
		// connected but no flexibility announced yet: standby, not disconnected
		return api.StatusB, nil
	}

	return ohpcfStatus(state), nil
}

// Enabled reports the commanded on/off intent; Status reflects the actual
// compressor state.
func (c *EEBusOHPCF) Enabled() (bool, error) {
	if _, err := c.ohpcf.Required(); err != nil {
		return false, err
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
func (c *EEBusOHPCF) stop() error {
	if pausable, err := c.ohpcf.Read(ucapi.CemOHPCFInterface.ConsumptionIsPausable); err == nil && pausable {
		return c.ohpcf.Write(ucapi.CemOHPCFInterface.PausePowerConsumptionProcess)
	}

	if stoppable, err := c.ohpcf.Read(ucapi.CemOHPCFInterface.ConsumptionIsStoppable); err == nil && stoppable {
		return c.ohpcf.Write(ucapi.CemOHPCFInterface.AbortPowerConsumptionProcess)
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
	limit, err := c.lpc.Read(ucapi.EgLPCInterface.ConsumptionLimit)
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
	// TODO: change api.Dimmer to make the limit configurable; use a fixed 0W safe limit for now
	return c.lpc.Write(func(uc ucapi.EgLPCInterface, entity spineapi.EntityRemoteInterface, cb eebus.ResultCB) (*model.MsgCounterType, error) {
		return uc.WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: 0, IsActive: dim}, cb)
	})
}

// apply issues the command to align the optional consumption with the on/off
// intent. It is idempotent: ohpcfControlAction only acts on a state transition.
func (c *EEBusOHPCF) apply() error {
	if _, err := c.ohpcf.Required(); err != nil {
		return err
	}

	state, err := c.ohpcf.Read(ucapi.CemOHPCFInterface.PowerConsumptionProcessState)
	if err != nil {
		// no process state announced yet, nothing to control
		return nil
	}

	action := ohpcfControlAction(state, c.lastEnabled())
	if action == ohpcfNone {
		return nil
	}

	switch action {
	case ohpcfSchedule:
		return c.ohpcf.Write(func(uc ucapi.CemOHPCFInterface, entity spineapi.EntityRemoteInterface, cb eebus.ResultCB) (*model.MsgCounterType, error) {
			// 0 = start immediately (relative schedule, see SchedulePowerConsumptionProcess)
			return uc.SchedulePowerConsumptionProcess(entity, 0, cb)
		})
	case ohpcfResume:
		return c.ohpcf.Write(ucapi.CemOHPCFInterface.ResumePowerConsumptionProcess)
	case ohpcfStop:
		return c.stop()
	}

	return nil
}

var _ api.PowerLimiter = (*EEBusOHPCF)(nil)

// GetMinMaxPower implements the api.PowerLimiter interface, reporting the
// optional consumption as expected min/max or ErrNotAvailable if none.
func (c *EEBusOHPCF) GetMinMaxPower() (float64, float64, error) {
	if _, err := c.ohpcf.Required(); err != nil {
		return 0, 0, err
	}

	if power, _ := c.ohpcf.Read(ucapi.CemOHPCFInterface.RequestedPowerEstimate); power > 0 {
		return power, power, nil
	}

	if power, _ := c.ohpcf.Read(ucapi.CemOHPCFInterface.RequestedPowerMax); power > 0 {
		return power, power, nil
	}

	return 0, 0, api.ErrNotAvailable
}

var _ api.Meter = (*EEBusOHPCF)(nil)

// CurrentPower implements the api.Meter interface and reports the heat pump's
// measured power consumption via the MPC use case.
func (c *EEBusOHPCF) CurrentPower() (float64, error) {
	return c.mpc.Read(ucapi.MaMPCInterface.Power)
}

var _ api.Battery = (*EEBusOHPCF)(nil)

// Soc implements the api.Battery interface and reports the heat pump's domestic
// hot water temperature in °C via the MDT use case.
func (c *EEBusOHPCF) Soc() (float64, error) {
	return c.mdt.Read(
		func(uc ucapi.MaMDTInterface, entity spineapi.EntityRemoteInterface) (float64, error) {
			return uc.Temperature(entity, model.UnitOfMeasurementTypedegC)
		})
}
