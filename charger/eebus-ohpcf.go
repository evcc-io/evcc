package charger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
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

	mux        sync.RWMutex
	log        *util.Logger
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
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
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
	inst, err := eebus.Instance()
	if err != nil {
		return nil, err
	}

	c := &EEBusOHPCF{
		embed:     embed,
		log:       util.NewLogger("eebus-ohpcf"),
		cem:       inst.CustomerEnergyManagement(),
		ma:        inst.MonitoringAppliance(),
		connector: eebus.NewConnector(),
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

// Enabled reports the commanded on/off intent; Status reflects the actual
// compressor state.
func (c *EEBusOHPCF) Enabled() (bool, error) {
	if _, ok := c.connectedCompressor(); !ok {
		return false, errNotConnected
	}

	return c.lastEnabled(), nil
}

// Enable schedules or resumes the optional consumption when on, pauses or
// aborts it when off.
func (c *EEBusOHPCF) Enable(enable bool) error {
	c.setEnabled(enable)

	return c.apply()
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
		return c.await(func(cb func(model.ResultDataType)) (*model.MsgCounterType, error) {
			return c.cem.OHPCF.PausePowerConsumptionProcess(entity, cb)
		})
	}

	if stoppable, err := c.cem.OHPCF.ConsumptionIsStoppable(entity); err == nil && stoppable {
		return c.await(func(cb func(model.ResultDataType)) (*model.MsgCounterType, error) {
			return c.cem.OHPCF.AbortPowerConsumptionProcess(entity, cb)
		})
	}

	return api.ErrNotAvailable
}

// ohpcfWriteTimeout bounds how long a control write waits for its result.
const ohpcfWriteTimeout = 10 * time.Second

// await runs a control write and waits for the heat pump's result, returning an
// error if the write is rejected or no result arrives within the timeout.
func (c *EEBusOHPCF) await(write func(func(model.ResultDataType)) (*model.MsgCounterType, error)) error {
	res := make(chan model.ResultDataType, 1)

	if _, err := write(func(r model.ResultDataType) { res <- r }); err != nil {
		return err
	}

	select {
	case r := <-res:
		if r.ErrorNumber != nil && *r.ErrorNumber != 0 {
			err := fmt.Errorf("write rejected: %d", *r.ErrorNumber)
			if r.Description != nil {
				err = fmt.Errorf("%w (%s)", err, *r.Description)
			}
			c.log.ERROR.Println(err)
			return err
		}
		return nil
	case <-time.After(ohpcfWriteTimeout):
		return errors.New("write result timeout")
	}
}

// MaxCurrent implements the api.Charger interface. OHPCF is on/off and cannot
// be modulated, so the offered current is ignored.
func (c *EEBusOHPCF) MaxCurrent(int64) error {
	return c.apply()
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
		return c.await(func(cb func(model.ResultDataType)) (*model.MsgCounterType, error) {
			// 0 = start immediately (relative schedule, see SchedulePowerConsumptionProcess)
			return c.cem.OHPCF.SchedulePowerConsumptionProcess(entity, 0, cb)
		})
	case ohpcfResume:
		return c.await(func(cb func(model.ResultDataType)) (*model.MsgCounterType, error) {
			return c.cem.OHPCF.ResumePowerConsumptionProcess(entity, cb)
		})
	case ohpcfStop:
		return c.stop(entity)
	}

	return nil
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
