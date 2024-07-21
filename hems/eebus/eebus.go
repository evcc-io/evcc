package eebus

import (
	"errors"
	"os"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cs/lpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

const (
	maxIdRequestTimespan         = time.Second * 120
	idleFactor                   = 0.6
	voltage              float64 = 230
)

type minMax struct {
	min, max float64
}

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
}

// LPC, LPP, MPC, MGCP

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

	for _, s := range c.uc.LPC.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("LPC RemoteEntitiesScenarios:", s.Scenarios)
	}

	for _, s := range c.uc.LPP.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("LPP RemoteEntitiesScenarios:", s.Scenarios)
	}

	for _, s := range c.uc.MPC.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("MPC RemoteEntitiesScenarios:", s.Scenarios)
	}

	for _, s := range c.uc.MGCP.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("MGCP RemoteEntitiesScenarios:", s.Scenarios)
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
	case lpc.DataUpdateLimit:
		c.dataUpdateLimit()
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

func (c *EEBus) dataUpdateLimit() {
	limit, err := c.uc.LPC.ConsumptionLimit()
	if err != nil {
		c.log.ERROR.Println(err)
		return
	}

	c.limit = limit
}
