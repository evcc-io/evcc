package meter

import (
	"errors"
	"os"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

type EEBus struct {
	ski string

	mux sync.Mutex
	log *util.Logger
	lp  loadpoint.API

	connected     bool
	connectedC    chan bool
	connectedTime time.Time

	uc    *eebus.UseCasesCS
	limit ucapi.LoadLimit

	power, energy      provider.Value[float64]
	voltages, currents provider.Value[[]float64]
}

// New creates an EEBus HEMS from generic config
func New(other map[string]interface{}, site site.API) (*EEBus, error) {
	var cc struct {
		Ski string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski)
}

// NewEEBus creates EEBus charger
func NewEEBus(ski string) (*EEBus, error) {
	log := util.NewLogger("eebus")

	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		ski:        ski,
		log:        log,
		connectedC: make(chan bool, 1),
		uc:         eebus.Instance.ControllableSystem(),
	}

	if err := eebus.Instance.RegisterDevice(ski, c); err != nil {
		return nil, err
	}

	if err := c.waitForConnection(); err != nil {
		return c, err
	}

	return c, nil
}

// waitForConnection wait for initial connection and returns an error on failure
func (c *EEBus) waitForConnection() error {
	timeout := time.After(90 * time.Second)
	for {
		select {
		case <-timeout:
			return os.ErrDeadlineExceeded
		case connected := <-c.connectedC:
			if connected {
				return nil
			}
		}
	}
}

// EEBUSDeviceInterface

func (c *EEBus) DeviceConnect() {
	c.log.TRACE.Println("connect ski:", c.ski)
	c.setConnected(true)
}

func (c *EEBus) DeviceDisconnect() {
	c.log.TRACE.Println("disconnect ski:", c.ski)
	c.setConnected(false)
}

// UseCase specific events
func (c *EEBus) UseCaseEventCB(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
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

// set wether the EVSE is connected
func (c *EEBus) setConnected(connected bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.connected = connected
	if connected && !c.connected {
		c.connectedTime = time.Now()
	}

	select {
	case c.connectedC <- connected:
	default:
	}
}

func (c *EEBus) isConnected() bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.connected
}

func (c *EEBus) Run() {
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
	if err != nil {
		return 0, 0, 0, err
	}
	if len(res) != 3 {
		return 0, 0, 0, errors.New("invalid phase currents")
	}
	return res[0], res[1], res[2], nil
}

func (c *EEBus) PhaseVoltages() (float64, float64, float64, error) {
	res, err := c.voltages.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(res) != 3 {
		return 0, 0, 0, errors.New("invalid phase voltages")
	}
	return res[0], res[1], res[2], nil
}
