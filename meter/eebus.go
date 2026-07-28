package meter

import (
	"context"
	"fmt"
	"math"
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

	maEntity    *eebus.Entity[measurements]
	egLpcEntity *eebus.Entity[ucapi.EgLPCInterface]
	egLppEntity *eebus.Entity[ucapi.EgLPPInterface]
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

	eg := inst.EnergyGuard()

	c := &EEBus{
		log:         util.NewLogger("eebus-" + useCase),
		ma:          ma,
		eg:          eg,
		mm:          mm,
		scenarios:   scenarios,
		connector:   eebus.NewConnector(),
		maEntity:    eebus.NewEntity(mm),
		egLpcEntity: eebus.NewEntity(eg.EgLPCInterface),
		egLppEntity: eebus.NewEntity(eg.EgLPPInterface),
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

var _ api.Meter = (*EEBus)(nil)

func (c *EEBus) CurrentPower() (float64, error) {
	return c.maEntity.Read(c.scenarios.power, measurements.Power)
}

var _ api.MeterEnergy = (*EEBus)(nil)

func (c *EEBus) TotalEnergy() (float64, error) {
	return c.maEntity.Read(c.scenarios.energy, measurements.EnergyConsumed)
}

func (c *EEBus) readPhases(scenario uint, update func(mm measurements, entity spineapi.EntityRemoteInterface) ([]float64, error)) (float64, float64, float64, error) {
	res, err := c.maEntity.Read(scenario, update)
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
	return c.readPhases(c.scenarios.currents, measurements.CurrentPerPhase)
}

var _ api.PhaseVoltages = (*EEBus)(nil)

func (c *EEBus) Voltages() (float64, float64, float64, error) {
	return c.readPhases(c.scenarios.voltages, measurements.VoltagePerPhase)
}

var _ api.Dimmer = (*EEBus)(nil)

// Dimmed implements the api.Dimmer interface
func (c *EEBus) Dimmed() (bool, error) {
	limit, err := c.egLpcEntity.Read(eebus.LPCLimit, ucapi.EgLPCInterface.ConsumptionLimit)
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

	entity := c.egLpcEntity.Get()

	if entity == nil || !c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(entity, eebus.LPCLimit) {
		return api.ErrNotAvailable
	}

	return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
		return c.eg.EgLPCInterface.WriteConsumptionLimit(entity, ucapi.LoadLimit{Value: value, IsActive: dim}, cb)
	})
}

var _ api.Curtailer = (*EEBus)(nil)

// CurtailedPercent implements the api.Curtailer interface
func (c *EEBus) CurtailedPercent() (int, error) {
	limit, err := c.egLppEntity.Read(eebus.LPPLimit, ucapi.EgLPPInterface.ProductionLimit)
	if err != nil {
		return 0, err
	}

	// production limits are negative watts, a positive value is invalid
	if !limit.IsActive || limit.Value > 0 {
		return 100, nil
	}

	// without a nominal reference the limit cannot be expressed as a percent
	nominal, err := c.egLppEntity.Read(eebus.LPPElectricalConnection, ucapi.EgLPPInterface.ProductionNominalMax)
	if err != nil || nominal <= 0 {
		return 0, api.ErrNotAvailable
	}

	// round, the watt conversion does not reproduce the written percent exactly
	return int(math.Round(-limit.Value / nominal * 100)), nil
}

// SetCurtailPercent implements the api.Curtailer interface
func (c *EEBus) SetCurtailPercent(percent int) error {
	curtail := percent < 100

	entity := c.egLppEntity.Get()

	if entity == nil || !c.eg.EgLPPInterface.IsScenarioAvailableAtEntity(entity, eebus.LPPLimit) {
		return api.ErrNotAvailable
	}

	// derive a proportional feed-in limit from the producer's nominal power
	// (limits are negative watts); fall back to a safe 0W limit if unavailable
	var value float64
	if curtail {
		if nominal, err := c.egLppEntity.Read(eebus.LPPElectricalConnection, ucapi.EgLPPInterface.ProductionNominalMax); err == nil && nominal > 0 {
			value = -float64(percent) / 100 * nominal
		}
	}

	return eebus.Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
		return c.eg.EgLPPInterface.WriteProductionLimit(entity, ucapi.LoadLimit{Value: value, IsActive: curtail}, cb)
	})
}
