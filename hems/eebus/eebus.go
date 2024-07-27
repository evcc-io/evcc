package eebus

import (
	"errors"
	"sync"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
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

	root api.Circuit

	status        status
	statusUpdated time.Time

	limit            *ucapi.LoadLimit // LPC-041
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

	root := circuit.Root()
	if root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}
	if !root.HasMeter() {
		return nil, errors.New("hems requires root circuit to have meter")
	}

	// create new root circuit for LPC
	lpc, err := circuit.New(util.NewLogger("lpc"), "eebus", 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	// wrap old root with new pc parent
	if err := root.Wrap(lpc); err != nil {
		return nil, err
	}
	site.SetCircuit(lpc)

	return NewEEBus(cc.Ski, lpc)
}

// NewEEBus creates EEBus charger
func NewEEBus(ski string, root api.Circuit) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:       util.NewLogger("eebus"),
		root:      root,
		uc:        eebus.Instance.ControllableSystem(),
		Connector: eebus.NewConnector(nil),
		heartbeat: provider.NewValue[struct{}](2 * time.Minute), // LPC-031
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

	c.log.TRACE.Println("status:", c.status)

	// check heartbeat
	_, heartbeatErr := c.heartbeat.Get()
	if heartbeatErr != nil && c.status != StatusFailsafe {
		// LPC-914/2
		c.log.WARN.Println("missing heartbeat- entering failsafe mode")
		c.setStatusAndLimit(StatusFailsafe, c.failsafeLimit)

		return nil
	}

	// TODO
	// status init
	// status Unlimited/controlled
	// status Unlimited/autonomous

	switch c.status {
	case StatusUnlimited:
		// LPC-914/1
		if c.limit != nil && c.limit.IsActive {
			c.log.WARN.Println("active consumption limit")
			c.setStatusAndLimit(StatusLimited, c.limit.Value)
		}

	case StatusLimited:
		// limit updated?
		if !c.limit.IsActive {
			c.log.WARN.Println("inactive consumption limit")
			c.setStatusAndLimit(StatusUnlimited, 0)
			break
		}

		c.setLimit(c.limit.Value)

		// LPC-914/1
		if d := c.limit.Duration; d > 0 && time.Since(c.statusUpdated) > d {
			c.limit = nil

			c.log.DEBUG.Println("limit duration exceeded- return to normal")
			c.setStatusAndLimit(StatusUnlimited, 0)
		}

	case StatusFailsafe:
		// LPC-914/2
		if d := c.failsafeDuration; heartbeatErr == nil && time.Since(c.statusUpdated) > d {
			c.log.DEBUG.Println("heartbeat returned and failsafe duration exceeded- return to normal")
			c.setStatusAndLimit(StatusUnlimited, 0)
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
	c.root.SetMaxPower(limit)
}
