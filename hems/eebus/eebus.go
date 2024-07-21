package eebus

import (
	"errors"
	"sync"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cs/lpc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	uc *eebus.UseCasesCS

	limit ucapi.LoadLimit
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
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:       util.NewLogger("eebus"),
		uc:        eebus.Instance.ControllableSystem(),
		Connector: eebus.NewConnector(nil),
	}

	if err := eebus.Instance.RegisterDevice(ski, c); err != nil {
		return nil, err
	}

	if err := c.Wait(90 * time.Second); err != nil {
		return c, err
	}

	for _, s := range c.uc.LPC.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("LPC RemoteEntitiesScenarios:", s.Scenarios)
	}

	for _, s := range c.uc.LPP.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("LPP RemoteEntitiesScenarios:", s.Scenarios)
	}

	for _, s := range c.uc.MGCP.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("MGCP RemoteEntitiesScenarios:", s.Scenarios)
	}

	return c, nil
}

var _ eebus.Device = (*EEBus)(nil)

// UseCaseEvent implements the eebus.Device interface
func (c *EEBus) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	switch event {
	case lpc.DataUpdateLimit:
		c.dataUpdateLimit()
	}
}

func (c *EEBus) Run() {
}

func (c *EEBus) dataUpdateLimit() {
	limit, err := c.uc.LPC.ConsumptionLimit()
	if err != nil {
		c.log.ERROR.Println(err)
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	c.limit = limit
}
