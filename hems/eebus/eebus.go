package eebus

import (
	"errors"
	"os"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
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

	uc *eebus.UseCasesCS
	ev spineapi.EntityRemoteInterface

	log *util.Logger
	lp  loadpoint.API

	connected     bool
	connectedC    chan bool
	connectedTime time.Time

	muxEntity sync.Mutex
	mux       sync.Mutex
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

func (c *EEBus) setEvEntity(entity spineapi.EntityRemoteInterface) {
	c.muxEntity.Lock()
	defer c.muxEntity.Unlock()

	c.ev = entity
}

func (c *EEBus) evEntity() spineapi.EntityRemoteInterface {
	c.muxEntity.Lock()
	defer c.muxEntity.Unlock()

	return c.ev
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
func (c *EEBus) UseCaseEventCB(device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	// EV
	case evcc.EvConnected:
		c.log.TRACE.Println("EV Connected")
		c.setEvEntity(entity)
	case evcc.EvDisconnected:
		c.log.TRACE.Println("EV Disconnected")
		c.setEvEntity(nil)
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
