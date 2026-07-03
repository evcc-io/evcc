package meter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

// EEBus is an EEBus meter implementation supporting MGCP, MPC, LPC and LPP use cases
// Uses MGCP (Monitoring of Grid Connection Point) only when usage="grid"
// Uses MPC (Monitoring & Power Consumption) for all other cases (default)
// Additionally supports LPC (Limitation of Power Consumption) and LPP (Limitation of Power Production)
type EEBus struct {
	log *util.Logger

	connector *eebus.Connector
	ma        *eebus.MonitoringAppliance
	eg        *eebus.EnergyGuard
	mm        measurements
	scenarios maScenarios

	mu          sync.Mutex
	maEntity    spineapi.EntityRemoteInterface
	egLpcEntity spineapi.EntityRemoteInterface
	egLppEntity spineapi.EntityRemoteInterface
}

// maScenarios holds the spec scenario numbers for the active monitoring use case.
// MGCP and MPC use different scenario numbers for the same physical quantity, so
// IsScenarioAvailableAtEntity must be called with the per-UC value.
type maScenarios struct {
	power    uint
	energy   uint
	currents uint
	voltages uint
}

var (
	mpcScenarios = maScenarios{
		power:    eebus.MPCPower,
		energy:   eebus.MPCEnergyConsumed,
		currents: eebus.MPCCurrentPerPhase,
		voltages: eebus.MPCVoltagePerPhase,
	}
	mgcpScenarios = maScenarios{
		power:    eebus.MGCPPower,
		energy:   eebus.MGCPEnergyConsumed,
		currents: eebus.MGCPCurrentPerPhase,
		voltages: eebus.MGCPVoltagePerPhase,
	}
)

type measurements interface {
	eebusapi.UseCaseBaseInterface
	Power(entity spineapi.EntityRemoteInterface) (float64, error)
	EnergyConsumed(entity spineapi.EntityRemoteInterface) (float64, error)
	CurrentPerPhase(entity spineapi.EntityRemoteInterface) ([]float64, error)
	VoltagePerPhase(entity spineapi.EntityRemoteInterface) ([]float64, error)
}

func init() {
	registry.AddCtx("eebus", NewEEBusFromConfig)
}

// NewEEBusFromConfig creates an EEBus meter from generic config
func NewEEBusFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	var cc struct {
		Ski, Ip  string
		Usage    *templates.Usage
		Timeout_ time.Duration `mapstructure:"timeout"` // TODO deprecated
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(ctx, cc.Ski, cc.Ip, cc.Usage)
}

// NewEEBus creates an EEBus meter
// Uses MGCP only when usage="grid", otherwise uses MPC (default)
func NewEEBus(ctx context.Context, ski, ip string, usage *templates.Usage) (api.Meter, error) {
	inst, err := eebus.Instance()
	if err != nil {
		return nil, err
	}

	ma := inst.MonitoringAppliance()

	// Use MGCP only for explicit grid usage, MPC for everything else (default)
	useCase := "mpc"
	mm := measurements(ma.MaMPCInterface)
	scenarios := mpcScenarios

	if usage != nil && *usage == templates.UsageGrid {
		useCase = "mgcp"
		mm = ma.MaMGCPInterface
		scenarios = mgcpScenarios
	}

	c := &EEBus{
		log:       util.NewLogger("eebus-" + useCase),
		ma:        ma,
		eg:        inst.EnergyGuard(),
		mm:        mm,
		scenarios: scenarios,
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

	// monitoring appliance
	eebus.LogEntities(c.log.DEBUG, "MA MPC", c.ma.MaMPCInterface)
	eebus.LogEntities(c.log.DEBUG, "MA MGCP", c.ma.MaMGCPInterface)

	// energy guard
	eebus.LogEntities(c.log.DEBUG, "EG LPC", c.eg.EgLPCInterface)
	eebus.LogEntities(c.log.DEBUG, "EG LPP", c.eg.EgLPPInterface)

	return c, nil
}

func eebusReadValue[T any](uc eebusapi.UseCaseBaseInterface, entity spineapi.EntityRemoteInterface, scenario uint, update func(entity spineapi.EntityRemoteInterface) (T, error)) (T, error) {
	var zero T

	if entity == nil || !uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return zero, api.ErrNotAvailable
	}

	res, err := update(entity)
	if err != nil {
		// scenario announced but no usable value yet
		if errors.Is(err, eebusapi.ErrDataNotAvailable) ||
			errors.Is(err, eebusapi.ErrMetadataNotAvailable) ||
			errors.Is(err, eebusapi.ErrDataInvalid) {
			err = api.ErrNotAvailable
		}
		return zero, err
	}

	return res, nil
}

func (c *EEBus) readValue(scenario uint, update func(entity spineapi.EntityRemoteInterface) (float64, error)) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return eebusReadValue(c.mm, c.maEntity, scenario, update)
}

var _ api.Meter = (*EEBus)(nil)

func (c *EEBus) CurrentPower() (float64, error) {
	return c.readValue(c.scenarios.power, c.mm.Power)
}

var _ api.MeterEnergy = (*EEBus)(nil)

func (c *EEBus) TotalEnergy() (float64, error) {
	return c.readValue(c.scenarios.energy, c.mm.EnergyConsumed)
}

func (c *EEBus) readPhases(scenario uint, update func(entity spineapi.EntityRemoteInterface) ([]float64, error)) (float64, float64, float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	res, err := eebusReadValue(c.mm, c.maEntity, scenario, update)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(res) == 0 {
		return 0, 0, 0, api.ErrNotAvailable
	}

	if len(res) > 3 {
		return 0, 0, 0, fmt.Errorf("invalid phases: %v", res)
	}

	for len(res) < 3 {
		res = append(res, 0)
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*EEBus)(nil)

func (c *EEBus) Currents() (float64, float64, float64, error) {
	return c.readPhases(c.scenarios.currents, c.mm.CurrentPerPhase)
}

var _ api.PhaseVoltages = (*EEBus)(nil)

func (c *EEBus) Voltages() (float64, float64, float64, error) {
	return c.readPhases(c.scenarios.voltages, c.mm.VoltagePerPhase)
}

var _ api.Dimmer = (*EEBus)(nil)

// Dimmed implements the api.Dimmer interface
func (c *EEBus) Dimmed() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit, err := eebusReadValue(c.eg.EgLPCInterface, c.egLpcEntity, eebus.LPCLimit, c.eg.EgLPCInterface.ConsumptionLimit)
	if err != nil {
		return false, err
	}

	// an active limit means dimmed; the applied limit value is 0W, so a
	// value-based check would never report the dimmed state and never release it
	return limit.IsActive, nil
}

// Dim implements the api.Dimmer interface
func (c *EEBus) Dim(dim bool) error {
	// Sets or removes the consumption power limit

	// TODO: change api.Dimmer to make limit configurable
	// For now, we use a fixed safe limit of 0W
	limit := 0.0

	var value float64
	if dim {
		value = limit
	}

	c.mu.Lock()
	entity := c.egLpcEntity
	c.mu.Unlock()

	if entity == nil || !c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(entity, eebus.LPCLimit) {
		return api.ErrNotAvailable
	}

	return eebus.Await(func(cb func(model.ResultDataType)) (*model.MsgCounterType, error) {
		return c.eg.EgLPCInterface.WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: value, IsActive: dim}, cb)
	})
}

var _ api.Curtailer = (*EEBus)(nil)

// Curtailed implements the api.Curtailer interface
func (c *EEBus) Curtailed() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit, err := eebusReadValue(c.eg.EgLPPInterface, c.egLppEntity, eebus.LPPLimit, c.eg.EgLPPInterface.ProductionLimit)
	if err != nil {
		return false, err
	}

	// Check if limit is active and has a valid power value (valid is zero or negative)
	return limit.IsActive && limit.Value <= 0, nil
}

// SetCurtailPercent implements the api.Curtailer interface
func (c *EEBus) SetCurtailPercent(percent int) error {
	curtail := percent < 100

	c.mu.Lock()
	entity := c.egLppEntity
	c.mu.Unlock()

	if entity == nil || !c.eg.EgLPPInterface.IsScenarioAvailableAtEntity(entity, eebus.LPPLimit) {
		return api.ErrNotAvailable
	}

	// derive a proportional feed-in limit from the producer's nominal power
	// (limits are negative watts); fall back to a safe 0W limit if unavailable
	var value float64
	if curtail {
		if nominal, err := c.eg.EgLPPInterface.ProductionNominalMax(entity); err == nil && nominal > 0 {
			value = -float64(percent) / 100 * nominal
		}
	}

	return eebus.Await(func(cb func(model.ResultDataType)) (*model.MsgCounterType, error) {
		return c.eg.EgLPPInterface.WriteProductionLimit(entity, ucapi.LoadLimit{Value: value, IsActive: curtail}, cb)
	})
}
