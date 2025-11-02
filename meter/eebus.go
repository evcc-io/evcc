package meter

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

// EEBus is an EEBus meter implementation supporting MGCP, MPC, and LPC use cases
// Uses MGCP (Monitoring of Grid Connection Point) only when usage="grid"
// Uses MPC (Monitoring & Power Consumption) for all other cases (default)
// Additionally supports LPC (Limitation of Power Consumption)
type EEBus struct {
	log *util.Logger

	*eebus.Connector
	ma *eebus.MonitoringAppliance
	eg *eebus.EnergyGuard
	mm measurements

	power    *util.Value[float64]
	energy   *util.Value[float64]
	currents *util.Value[[]float64]
	voltages *util.Value[[]float64]

	// TODO use util.Value
	mu               sync.Mutex
	consumptionLimit *ucapi.LoadLimit
	egLpcEntity      spineapi.EntityRemoteInterface
	// failsafeLimit    float64
	// failsafeDuration time.Duration
}

type measurements interface {
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
	cc := struct {
		Ski     string
		Ip      string
		Usage   *templates.Usage
		Timeout time.Duration
	}{
		Timeout: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(ctx, cc.Ski, cc.Ip, cc.Usage, cc.Timeout)
}

// NewEEBus creates an EEBus meter
// Uses MGCP only when usage="grid", otherwise uses MPC (default)
func NewEEBus(ctx context.Context, ski, ip string, usage *templates.Usage, timeout time.Duration) (api.Meter, error) {
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
		power:     util.NewValue[float64](timeout),
		energy:    util.NewValue[float64](timeout),
		currents:  util.NewValue[[]float64](timeout),
		voltages:  util.NewValue[[]float64](timeout),
	}

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	return c, nil
}

var _ eebus.Device = (*EEBus)(nil)

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	c.log.TRACE.Printf("recv: %s", event)

	switch event {
	// Monitoring Appliance
	case mpc.DataUpdatePower, mgcp.DataUpdatePower:
		c.maDataUpdatePower(entity)
	case mpc.DataUpdateEnergyConsumed, mgcp.DataUpdateEnergyConsumed:
		c.maDataUpdateEnergyConsumed(entity)
	case mpc.DataUpdateCurrentsPerPhase, mgcp.DataUpdateCurrentPerPhase:
		c.maDataUpdateCurrentPerPhase(entity)
	case mpc.DataUpdateVoltagePerPhase, mgcp.DataUpdateVoltagePerPhase:
		c.maDataUpdateVoltagePerPhase(entity)

	// Energy Guard
	case lpc.UseCaseSupportUpdate:
		c.egLpcUseCaseSupportUpdate(entity)
	case lpc.DataUpdateLimit:
		c.egLpcDataUpdateLimit(entity)
	}
}

func (c *EEBus) maDataUpdatePower(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.Power(entity)
	if err != nil {
		c.log.ERROR.Println("Power:", err)
		return
	}
	c.log.TRACE.Printf("Power: %.0fW", data)
	c.power.Set(data)
}

func (c *EEBus) maDataUpdateEnergyConsumed(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.EnergyConsumed(entity)
	if err != nil {
		c.log.ERROR.Println("EnergyConsumed:", err)
		return
	}
	c.log.TRACE.Printf("EnergyConsumed: %.1fkWh", data/1000)
	// Convert Wh to kWh
	c.energy.Set(data / 1000)
}

func (c *EEBus) maDataUpdateCurrentPerPhase(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.CurrentPerPhase(entity)
	if err != nil {
		c.log.ERROR.Println("CurrentPerPhase:", err)
		return
	}
	c.currents.Set(data)
}

func (c *EEBus) maDataUpdateVoltagePerPhase(entity spineapi.EntityRemoteInterface) {
	data, err := c.mm.VoltagePerPhase(entity)
	if err != nil {
		c.log.ERROR.Println("VoltagePerPhase:", err)
		return
	}
	c.voltages.Set(data)
}

var _ api.Meter = (*EEBus)(nil)

func (c *EEBus) CurrentPower() (float64, error) {
	return c.power.Get()
}

var _ api.MeterEnergy = (*EEBus)(nil)

func (c *EEBus) TotalEnergy() (float64, error) {
	res, err := c.energy.Get()
	if err != nil {
		return 0, api.ErrNotAvailable
	}

	return res, nil
}

var _ api.PhaseCurrents = (*EEBus)(nil)

func (c *EEBus) Currents() (float64, float64, float64, error) {
	res, err := c.currents.Get()
	if err != nil {
		return 0, 0, 0, api.ErrNotAvailable
	}
	if len(res) != 3 {
		return 0, 0, 0, errors.New("invalid phase currents")
	}
	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*EEBus)(nil)

func (c *EEBus) Voltages() (float64, float64, float64, error) {
	res, err := c.voltages.Get()
	if err != nil {
		return 0, 0, 0, api.ErrNotAvailable
	}
	if len(res) != 3 {
		return 0, 0, 0, errors.New("invalid phase voltages")
	}
	return res[0], res[1], res[2], nil
}

//
// Energy Guard
//

func (c *EEBus) egLpcUseCaseSupportUpdate(entity spineapi.EntityRemoteInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.egLpcEntity = entity
}

func (c *EEBus) egLpcDataUpdateLimit(entity spineapi.EntityRemoteInterface) {
	limit, err := c.eg.EgLPCInterface.ConsumptionLimit(entity)
	if err != nil {
		c.log.ERROR.Println("EG LPC ConsumptionLimit:", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.consumptionLimit = &limit
}

var _ api.Dimmer = (*EEBus)(nil)

// Dimmed implements the api.Dimmer interface
func (c *EEBus) Dimmed() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if limit is active and has a valid power value
	return c.consumptionLimit != nil && c.consumptionLimit.IsActive && c.consumptionLimit.Value > 0, nil
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

	if c.egLpcEntity == nil {
		return api.ErrNotAvailable
	}

	if !slices.Contains(c.eg.EgLPCInterface.AvailableScenariosForEntity(c.egLpcEntity), 1) {
		return errors.New("scenario 1 not supported")
	}

	_, err := c.eg.EgLPCInterface.WriteConsumptionLimit(c.egLpcEntity, ucapi.LoadLimit{
		Value:    value,
		IsActive: dim,
	}, func(result model.ResultDataType) {
		if result.ErrorNumber != nil {
			c.log.ERROR.Println("ErrorNumber", *result.ErrorNumber)
		}
		if result.Description != nil {
			c.log.ERROR.Println("Description", *result.Description)
		}
	})

	return err
}
