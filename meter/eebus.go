package meter

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	*eebus.Connector
	ma *eebus.MonitoringAppliance
	eg *eebus.EnergyGuard
	mm measurements

	mu          sync.Mutex
	maEntity    spineapi.EntityRemoteInterface
	egLpcEntity spineapi.EntityRemoteInterface
	egLppEntity spineapi.EntityRemoteInterface
}

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
		Ski      string
		Ip       string
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
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	ma := eebus.Instance.MonitoringAppliance()

	// Use MGCP only for explicit grid usage, MPC for everything else (default)
	useCase := "mpc"
	mm := measurements(ma.MaMPCInterface)

	if usage != nil && *usage == templates.UsageGrid {
		useCase = "mgcp"
		mm = ma.MaMGCPInterface
	}

	c := &EEBus{
		log:       util.NewLogger("eebus-" + useCase),
		ma:        ma,
		eg:        eebus.Instance.EnergyGuard(),
		mm:        mm,
		Connector: eebus.NewConnector(),
	}

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	// monitoring appliance
	eebus.LogEntities(c.log.DEBUG, "MA MPC", c.ma.MaMPCInterface)
	eebus.LogEntities(c.log.DEBUG, "MA MGCP", c.ma.MaMGCPInterface)

	// energy guard
	eebus.LogEntities(c.log.DEBUG, "EG LPC", c.eg.EgLPCInterface)
	eebus.LogEntities(c.log.DEBUG, "EG LPP", c.eg.EgLPPInterface)

	return c, nil
}

func eebusReadValue[T any](scenario uint, uc eebusapi.UseCaseBaseInterface, entity spineapi.EntityRemoteInterface, update func(entity spineapi.EntityRemoteInterface) (T, error)) (T, error) {
	var zero T

	if entity == nil || !uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return zero, api.ErrNotAvailable
	}

	res, err := update(entity)
	if err != nil {
		// announced but not provided
		if errors.Is(err, eebusapi.ErrDataNotAvailable) {
			err = api.ErrNotAvailable
		}
		return zero, err
	}

	return res, nil
}

func (c *EEBus) readValue(scenario uint, update func(entity spineapi.EntityRemoteInterface) (float64, error)) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return eebusReadValue(scenario, c.mm, c.maEntity, update)
}

var _ api.Meter = (*EEBus)(nil)

func (c *EEBus) CurrentPower() (float64, error) {
	return c.readValue(1, c.mm.Power)
}

var _ api.MeterEnergy = (*EEBus)(nil)

func (c *EEBus) TotalEnergy() (float64, error) {
	return c.readValue(2, c.mm.EnergyConsumed)
}

func (c *EEBus) readPhases(scenario uint, update func(entity spineapi.EntityRemoteInterface) ([]float64, error)) (float64, float64, float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	res, err := eebusReadValue(scenario, c.mm, c.maEntity, update)
	if err != nil {
		// announced but not provided
		if errors.Is(err, eebusapi.ErrDataNotAvailable) {
			err = api.ErrNotAvailable
		}
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
	return c.readPhases(3, c.mm.CurrentPerPhase)
}

var _ api.PhaseVoltages = (*EEBus)(nil)

func (c *EEBus) Voltages() (float64, float64, float64, error) {
	return c.readPhases(4, c.mm.VoltagePerPhase)
}

var _ api.Dimmer = (*EEBus)(nil)

// Dimmed implements the api.Dimmer interface
func (c *EEBus) Dimmed() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit, err := eebusReadValue(1, c.eg.EgLPCInterface, c.egLpcEntity, c.eg.EgLPCInterface.ConsumptionLimit)
	if err != nil {
		return false, err
	}

	// Check if limit is active and has a valid power value
	return limit.IsActive && limit.Value > 0, nil
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
	defer c.mu.Unlock()

	if c.egLpcEntity == nil || !c.eg.EgLPCInterface.IsScenarioAvailableAtEntity(c.egLpcEntity, 1) {
		return api.ErrNotAvailable
	}

	_, err := c.eg.EgLPCInterface.WriteConsumptionLimit(c.egLpcEntity, ucapi.LoadLimit{
		Value:    value,
		IsActive: dim,
	}, c.callbackResult("consumption limit"))

	return err
}

var _ api.Curtailer = (*EEBus)(nil)

// Curtailed implements the api.Curtailer interface
func (c *EEBus) Curtailed() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit, err := eebusReadValue(1, c.eg.EgLPPInterface, c.egLppEntity, c.eg.EgLPPInterface.ProductionLimit)
	if err != nil {
		return false, err
	}

	// Check if limit is active and has a valid power value
	return limit.IsActive && limit.Value > 0, nil
}

// Curtail implements the api.Curtailer interface
func (c *EEBus) Curtail(curtail bool) error {
	// Sets or removes the production power limit

	// TODO: change api.Curtailer to make limit configurable
	// For now, we use a fixed safe limit of 0W
	limit := 0.0

	var value float64
	if curtail {
		value = limit
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.egLppEntity == nil || !c.eg.EgLPPInterface.IsScenarioAvailableAtEntity(c.egLppEntity, 1) {
		return api.ErrNotAvailable
	}

	_, err := c.eg.EgLPPInterface.WriteProductionLimit(c.egLppEntity, ucapi.LoadLimit{
		Value:    value,
		IsActive: curtail,
	}, c.callbackResult("production limit"))

	return err
}

func (c *EEBus) callbackResult(typ string) func(result model.ResultDataType) {
	return func(result model.ResultDataType) {
		sb := new(strings.Builder)

		if result.ErrorNumber != nil {
			fmt.Fprint(sb, *result.ErrorNumber)
		}
		if result.Description != nil {
			if sb.Len() > 0 {
				fmt.Print(sb, ":")
			}
			fmt.Print(sb, *result.Description)
		}
		if sb.Len() > 0 {
			c.log.ERROR.Printf("%s: %s", typ, sb.String())
		}
	}
}
