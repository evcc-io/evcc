package eebus

import (
	"errors"
	"sync"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	uc *eebus.UseCasesCS

	status        status
	statusUpdated time.Time

	limit        *ucapi.LoadLimit
	limitUpdated time.Time

	failsafeLimit    float64
	failsafeDuration time.Duration

	heartbeat *provider.Value[struct{}]
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
		heartbeat: provider.NewValue[struct{}](time.Minute), // TODO spec
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

func (c *EEBus) Run() {
	// TODO check interval
	for range time.Tick(10 * time.Second) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

// TODO check state machine against spec
func (c *EEBus) run() error {
	c.mux.RLock()
	defer c.mux.RUnlock()

	// check heartbeat
	_, heartbeatErr := c.heartbeat.Get()
	if heartbeatErr != nil && c.status != StatusFailsafe {
		c.log.WARN.Println("missing heartbeat- entering failsafe mode")
		c.setStatusAndLimit(StatusFailsafe, c.failsafeLimit)

		return nil
	}

	switch c.status {
	case StatusOK:
		if c.limit != nil {
			c.log.WARN.Println("active consumption limit")
			c.setStatusAndLimit(StatusLimit, c.limit.Value)
		}

	case StatusLimit:
		// limit updated?
		c.setLimit(c.limit.Value)

		if d := c.limit.Duration; time.Since(c.statusUpdated) > d {
			c.limit = nil

			c.log.DEBUG.Println("limit duration exceeded- return to normal")
			c.setStatusAndLimit(StatusOK, 0)
		}

	case StatusFailsafe:
		if d := c.failsafeDuration; heartbeatErr == nil && time.Since(c.statusUpdated) > d {
			c.log.DEBUG.Println("heartbeat returned and failsafe duration exceeded- return to normal")
			c.setStatusAndLimit(StatusOK, 0)
		}
	}

	return nil
}

func (c *EEBus) setStatusAndLimit(status status, limit float64) {
	c.status = status
	c.statusUpdated = time.Now()

	c.setLimit(limit)
}

func (c *EEBus) setLimit(limit float64) {
	// TODO update root circuit
}
