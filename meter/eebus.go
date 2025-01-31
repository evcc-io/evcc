package meter

import (
	"errors"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

type EEBus struct {
	log *util.Logger

	*eebus.Connector
	uc *eebus.UseCasesCS

	power, energy      *util.Value[float64]
	voltages, currents *util.Value[[]float64]
}

func init() {
	registry.Add("eebus", NewEEBusFromConfig)
}

// New creates an EEBus HEMS from generic config
func NewEEBusFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Ski     string
		Ip      string
		Timeout time.Duration
	}{
		Timeout: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski, cc.Ip, cc.Timeout)
}

// NewEEBus creates EEBus charger
func NewEEBus(ski, ip string, timeout time.Duration) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:       util.NewLogger("eebus"),
		uc:        eebus.Instance.ControllableSystem(),
		Connector: eebus.NewConnector(),
		power:     util.NewValue[float64](timeout),
		energy:    util.NewValue[float64](timeout),
		voltages:  util.NewValue[[]float64](timeout),
		currents:  util.NewValue[[]float64](timeout),
	}

	if err := eebus.Instance.RegisterDevice(ski, ip, c); err != nil {
		return nil, err
	}

	if err := c.Wait(90 * time.Second); err != nil {
		return c, err
	}

	return c, nil
}

var _ eebus.Device = (*EEBus)(nil)

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
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
}

func (c *EEBus) dataUpdatePower(entity spineapi.EntityRemoteInterface) {
	data, err := c.uc.MGCP.Power(entity)
	if err != nil {
		c.log.ERROR.Println("MGCP.Power:", err)
		return
	}
	c.power.Set(data)
}

func (c *EEBus) dataUpdateEnergyConsumed(entity spineapi.EntityRemoteInterface) {
	data, err := c.uc.MGCP.EnergyConsumed(entity)
	if err != nil {
		c.log.ERROR.Println("MGCP.EnergyConsumed:", err)
		return
	}
	c.energy.Set(data)
}

func (c *EEBus) dataUpdateCurrentPerPhase(entity spineapi.EntityRemoteInterface) {
	data, err := c.uc.MGCP.CurrentPerPhase(entity)
	if err != nil {
		c.log.ERROR.Println("MGCP.CurrentPerPhase:", err)
		return
	}
	c.currents.Set(data)
}

func (c *EEBus) dataUpdateVoltagePerPhase(entity spineapi.EntityRemoteInterface) {
	data, err := c.uc.MGCP.VoltagePerPhase(entity)
	if err != nil {
		c.log.ERROR.Println("MGCP.VoltagePerPhase:", err)
		return
	}
	c.voltages.Set(data)
}

func (c *EEBus) CurrentPower() (float64, error) {
	return c.power.Get()
}

func (c *EEBus) TotalEnergy() (float64, error) {
	return c.energy.Get()
}

func (c *EEBus) PhaseCurrents() (float64, float64, float64, error) {
	res, err := c.currents.Get()
	if err == nil && len(res) != 3 {
		err = errors.New("invalid phase currents")
	}
	if err != nil {
		return 0, 0, 0, err
	}
	return res[0], res[1], res[2], nil
}

func (c *EEBus) PhaseVoltages() (float64, float64, float64, error) {
	res, err := c.voltages.Get()
	if err == nil && len(res) != 3 {
		err = errors.New("invalid phase voltages")
	}
	if err != nil {
		return 0, 0, 0, err
	}
	return res[0], res[1], res[2], nil
}
