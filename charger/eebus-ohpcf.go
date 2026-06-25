package charger

import (
	"context"
	"errors"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
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

	mux        sync.RWMutex
	log        *util.Logger
	lp         loadpoint.API
	compressor spineapi.EntityRemoteInterface
	mpcEntity  spineapi.EntityRemoteInterface
	enabled    bool

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
		embed `mapstructure:",squash"`
		Ski   string
		Ip    string
	}{
		embed: embed{
			Features_: []api.Feature{api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBusOHPCF(ctx, &cc.embed, cc.Ski, cc.Ip)
}

// NewEEBusOHPCF creates an EEBus OHPCF charger, registers it with the EEBus
// instance and waits for the connection.
func NewEEBusOHPCF(ctx context.Context, embed *embed, ski, ip string) (api.Charger, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBusOHPCF{
		embed:     embed,
		log:       util.NewLogger("eebus-ohpcf"),
		cem:       eebus.Instance.CustomerEnergyManagement(),
		ma:        eebus.Instance.MonitoringAppliance(),
		connector: eebus.NewConnector(),
	}

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.connector.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	// unregister device when context is cancelled (e.g. UI config validation)
	go func() {
		<-ctx.Done()
		eebus.Instance.UnregisterDevice(ski, c)
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

	c.mux.Lock()
	defer c.mux.Unlock()

	c.compressor = nil
	c.mpcEntity = nil
}

// UseCaseEvent implements the eebus.Device interface
func (c *EEBusOHPCF) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	// device/entity removal fires the use case update event with a nil entity
	if entity == nil {
		return
	}

	switch event {
	case ohpcf.UseCaseSupportUpdate,
		ohpcf.DataUpdateRequestedPowerEstimate,
		ohpcf.DataUpdateRequestedPowerMax,
		ohpcf.DataUpdateConsumptionIsStoppable,
		ohpcf.DataUpdateConsumptionIsPausable,
		ohpcf.DataUpdateConsumptionStartTime,
		ohpcf.DataUpdateConsumptionState,
		ohpcf.DataUpdateMinimalRunDuration,
		ohpcf.DataUpdateMinimalPauseDuration:
		c.mux.Lock()
		c.compressor = entity
		c.mux.Unlock()

	// Monitoring Appliance MPC provides the measured power consumption
	case mpc.UseCaseSupportUpdate:
		c.mux.Lock()
		// use most specific selector
		if c.mpcEntity == nil || len(entity.Address().Entity) < len(c.mpcEntity.Address().Entity) {
			c.mpcEntity = entity
		}
		c.mux.Unlock()
	}
}

func (c *EEBusOHPCF) connectedCompressor() (spineapi.EntityRemoteInterface, bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.compressor, c.compressor != nil
}

func (c *EEBusOHPCF) setEnabled(enabled bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.enabled = enabled
}

func (c *EEBusOHPCF) lastEnabled() bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.enabled
}

// ohpcfStatus maps the compressor power consumption process state to a charge status.
// running is active consumption (C), available/scheduled/paused mean the
// flexibility is present but not consuming (B), everything else (completed,
// stopped/aborted, no data) means there is nothing to control (A).
func ohpcfStatus(state ucapi.CompressorPowerConsumptionStateType) api.ChargeStatus {
	switch state {
	case ucapi.CompressorPowerConsumptionStateRunning:
		return api.StatusC
	case ucapi.CompressorPowerConsumptionStateAvailable,
		ucapi.CompressorPowerConsumptionStateScheduled,
		ucapi.CompressorPowerConsumptionStatePaused:
		return api.StatusB
	default:
		return api.StatusA
	}
}

// ohpcfEnabled reports whether the optional consumption has been committed.
// scheduled is treated as enabled (we asked it to run) so that the loadpoint
// does not re-issue the command while the compressor spins up to running.
func ohpcfEnabled(state ucapi.CompressorPowerConsumptionStateType) bool {
	return state == ucapi.CompressorPowerConsumptionStateScheduled ||
		state == ucapi.CompressorPowerConsumptionStateRunning
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
		return api.StatusA, nil
	}

	return ohpcfStatus(state), nil
}

// Enabled implements the api.Charger interface
func (c *EEBusOHPCF) Enabled() (bool, error) {
	entity, ok := c.connectedCompressor()
	if !ok {
		return false, errNotConnected
	}

	state, err := c.cem.OHPCF.PowerConsumptionProcessState(entity)
	if err != nil {
		return c.lastEnabled(), nil
	}

	return ohpcfEnabled(state), nil
}

// Enable implements the api.Charger interface.
// It records the on/off intent and pauses/aborts the optional consumption on
// disable. The actual start is deferred to MaxCurrent, which schedules the
// process once the available surplus power covers the compressor's request.
func (c *EEBusOHPCF) Enable(enable bool) error {
	c.setEnabled(enable)

	entity, ok := c.connectedCompressor()
	if !ok {
		return errNotConnected
	}

	if enable {
		return nil
	}

	state, err := c.cem.OHPCF.PowerConsumptionProcessState(entity)
	if err != nil {
		return nil
	}

	if ohpcfControlAction(state, false) == ohpcfStop {
		return c.stop(entity)
	}

	return nil
}

type ohpcfAction int

const (
	ohpcfNone ohpcfAction = iota
	ohpcfSchedule
	ohpcfResume
	ohpcfStop
)

// ohpcfControlAction decides which control command to issue given the current
// process state and whether the available surplus power is sufficient. It only
// returns an action when a state transition is required, so repeated calls with
// an unchanged state issue no further commands.
func ohpcfControlAction(state ucapi.CompressorPowerConsumptionStateType, sufficient bool) ohpcfAction {
	if sufficient {
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
		_, err := c.cem.OHPCF.PausePowerConsumptionProcess(entity, nil)
		return err
	}

	if stoppable, err := c.cem.OHPCF.ConsumptionIsStoppable(entity); err == nil && stoppable {
		_, err := c.cem.OHPCF.AbortPowerConsumptionProcess(entity, nil)
		return err
	}

	return api.ErrNotAvailable
}

// MaxCurrent implements the api.Charger interface.
// The compressor cannot be modulated (OHPCF has no power setpoint), but the
// loadpoint's allotted current still tells us how much surplus power is
// available. Once that covers the compressor's requested power, the optional
// consumption is scheduled to start now; otherwise it is paused.
func (c *EEBusOHPCF) MaxCurrent(current int64) error {
	entity, ok := c.connectedCompressor()
	if !ok {
		return errNotConnected
	}

	if !c.lastEnabled() {
		return nil
	}

	available := float64(current) * voltage * float64(c.phases())

	return c.applyAvailablePower(entity, available)
}

// phases returns the loadpoint's active phases, defaulting to single phase
func (c *EEBusOHPCF) phases() int {
	c.mux.RLock()
	lp := c.lp
	c.mux.RUnlock()

	if lp != nil {
		if p := lp.GetPhases(); p > 0 {
			return p
		}
	}

	return 1
}

// applyAvailablePower schedules or pauses the optional consumption depending on
// whether the available surplus power covers the compressor's requested power.
func (c *EEBusOHPCF) applyAvailablePower(entity spineapi.EntityRemoteInterface, available float64) error {
	requested, err := c.cem.OHPCF.RequestedPowerMax(entity)
	if err != nil {
		// no power request announced yet, nothing to schedule
		return nil
	}

	state, err := c.cem.OHPCF.PowerConsumptionProcessState(entity)
	if err != nil {
		return nil
	}

	switch ohpcfControlAction(state, available >= requested) {
	case ohpcfSchedule:
		_, err = c.cem.OHPCF.SchedulePowerConsumptionProcess(entity, time.Now(), nil)
	case ohpcfResume:
		_, err = c.cem.OHPCF.ResumePowerConsumptionProcess(entity, nil)
	case ohpcfStop:
		err = c.stop(entity)
	}

	return err
}

var _ loadpoint.Controller = (*EEBusOHPCF)(nil)

// LoadpointControl implements the loadpoint.Controller interface
func (c *EEBusOHPCF) LoadpointControl(lp loadpoint.API) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.lp = lp
}

var _ api.Meter = (*EEBusOHPCF)(nil)

// CurrentPower implements the api.Meter interface and reports the heat pump's
// measured power consumption via the MPC use case.
func (c *EEBusOHPCF) CurrentPower() (float64, error) {
	c.mux.RLock()
	entity := c.mpcEntity
	c.mux.RUnlock()

	if entity == nil || !c.ma.MaMPCInterface.IsScenarioAvailableAtEntity(entity, eebus.MPCPower) {
		return 0, api.ErrNotAvailable
	}

	power, err := c.ma.MaMPCInterface.Power(entity)
	if err != nil {
		return 0, eebus.WrapError(err)
	}

	return power, nil
}
