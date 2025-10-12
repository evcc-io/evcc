package meter

import (
	"context"
	"errors"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

// EEBus is an EEBus meter implementation supporting MGCP, MPC, and LPC use cases
// Uses MGCP (Monitoring of Grid Connection Point) for usage=grid
// Uses MPC (Monitoring & Power Consumption) for all other usages
// Additionally supports LPC (Limitation of Power Consumption)
type EEBus struct {
	log *util.Logger

	*eebus.Connector
	uc *eebus.UseCasesCS

	useMGCP bool // true for grid usage (MGCP), false for others (MPC)

	power    *util.Value[float64]
	energy   *util.Value[float64]
	currents *util.Value[[]float64]
	voltages *util.Value[[]float64]
}

func init() {
	registry.AddCtx("eebus", NewEEBusFromConfig)
}

// NewEEBusFromConfig creates an EEBus meter from generic config
func NewEEBusFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Ski     string
		Ip      string
		Usage   templates.Usage
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
// Uses MGCP for usage="grid", MPC for all other usages
func NewEEBus(ctx context.Context, ski, ip string, usage templates.Usage, timeout time.Duration) (api.Meter, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus is not configured yet. check config regarding cert, keys etc.")
	}

	// Use MGCP for grid connection points, MPC for everything else
	useMGCP := usage == templates.UsageGrid

	useCase := "mpc"
	if useMGCP {
		useCase = "mgcp"
	}

	c := &EEBus{
		log:       util.NewLogger("eebus-" + useCase),
		uc:        eebus.Instance.ControllableSystem(),
		Connector: eebus.NewConnector(),
		useMGCP:   useMGCP,
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
	c.log.TRACE.Printf("received event: %s", event)

	if c.useMGCP {
		// MGCP events for grid usage
		switch event {
		case mgcp.DataUpdatePower:
			c.dataUpdatePower(entity)
		case mgcp.DataUpdateEnergyConsumed:
			c.dataUpdateEnergyConsumed(entity)
		case mgcp.DataUpdateCurrentPerPhase:
			c.dataUpdateCurrentPerPhase(entity)
		case mgcp.DataUpdateVoltagePerPhase:
			c.dataUpdateVoltagePerPhase(entity)
		}
	} else {
		// MPC events for all other usages
		switch event {
		case mpc.DataUpdatePower:
			c.dataUpdatePower(entity)
		case mpc.DataUpdateEnergyConsumed:
			c.dataUpdateEnergyConsumed(entity)
		case mpc.DataUpdateCurrentsPerPhase:
			c.dataUpdateCurrentPerPhase(entity)
		case mpc.DataUpdateVoltagePerPhase:
			c.dataUpdateVoltagePerPhase(entity)
		}
	}
}

func (c *EEBus) dataUpdatePower(entity spineapi.EntityRemoteInterface) {
	var data float64
	var err error

	if c.useMGCP {
		data, err = c.uc.MGCP.Power(entity)
		if err != nil {
			c.log.ERROR.Println("MGCP.Power:", err)
			return
		}
		c.log.TRACE.Printf("MGCP.Power: %.0fW", data)
	} else {
		data, err = c.uc.MPC.Power(entity)
		if err != nil {
			c.log.ERROR.Println("MPC.Power:", err)
			return
		}
		c.log.TRACE.Printf("MPC.Power: %.0fW", data)
	}

	c.power.Set(data)
}

func (c *EEBus) dataUpdateEnergyConsumed(entity spineapi.EntityRemoteInterface) {
	var data float64
	var err error

	if c.useMGCP {
		data, err = c.uc.MGCP.EnergyConsumed(entity)
		if err != nil {
			c.log.ERROR.Println("MGCP.EnergyConsumed:", err)
			return
		}
		c.log.TRACE.Printf("MGCP.EnergyConsumed: %.1fkWh", data/1000)
	} else {
		data, err = c.uc.MPC.EnergyConsumed(entity)
		if err != nil {
			c.log.ERROR.Println("MPC.EnergyConsumed:", err)
			return
		}
		c.log.TRACE.Printf("MPC.EnergyConsumed: %.1fkWh", data/1000)
	}

	// Convert Wh to kWh
	c.energy.Set(data / 1000)
}

func (c *EEBus) dataUpdateCurrentPerPhase(entity spineapi.EntityRemoteInterface) {
	var data []float64
	var err error

	if c.useMGCP {
		data, err = c.uc.MGCP.CurrentPerPhase(entity)
		if err != nil {
			c.log.ERROR.Println("MGCP.CurrentPerPhase:", err)
			return
		}
	} else {
		data, err = c.uc.MPC.CurrentPerPhase(entity)
		if err != nil {
			c.log.ERROR.Println("MPC.CurrentPerPhase:", err)
			return
		}
	}

	c.currents.Set(data)
}

func (c *EEBus) dataUpdateVoltagePerPhase(entity spineapi.EntityRemoteInterface) {
	var data []float64
	var err error

	if c.useMGCP {
		data, err = c.uc.MGCP.VoltagePerPhase(entity)
		if err != nil {
			c.log.ERROR.Println("MGCP.VoltagePerPhase:", err)
			return
		}
	} else {
		data, err = c.uc.MPC.VoltagePerPhase(entity)
		if err != nil {
			c.log.ERROR.Println("MPC.VoltagePerPhase:", err)
			return
		}
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

var _ api.Dimmer = (*EEBus)(nil)

// Dimmed implements the api.Dimmer interface
func (c *EEBus) Dimmed() (bool, error) {
	limit, err := c.uc.LPC.ConsumptionLimit()
	if err != nil {
		// No limit available means not dimmed
		return false, nil
	}

	// Check if limit is active and has a valid power value
	return limit.IsActive && limit.Value > 0, nil
}

// Dim implements the api.Dimmer interface
func (c *EEBus) Dim(dim bool) error {
	// Sets or removes the consumption power limit
	// When dim=true, a default limit is set (4200W per §14a EnWG)
	// When dim=false, the limit is removed

	var limit float64
	if dim {
		limit = 4200
	}

	return c.uc.LPC.SetConsumptionLimit(ucapi.LoadLimit{
		Value:    limit,
		IsActive: dim,
	})
}
