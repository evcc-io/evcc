package eebus

import (
	"context"
	"errors"
	"sync"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	uc *eebus.UseCasesCS

	root api.Circuit

	status        status
	statusUpdated time.Time

	consumptionLimit *ucapi.LoadLimit // LPC-041
	failsafeLimit    float64
	failsafeDuration time.Duration

	heartbeat *util.Value[struct{}]
}

type Limits struct {
	ContractualConsumptionNominalMax    float64
	ConsumptionLimit                    float64
	FailsafeConsumptionActivePowerLimit float64
	FailsafeDurationMinimum             time.Duration
}

// New creates an EEBus HEMS from generic config
func New(ctx context.Context, other map[string]interface{}, site site.API) (*EEBus, error) {
	cc := struct {
		Ski    string
		Limits `mapstructure:",squash"`
	}{
		Limits: Limits{
			ContractualConsumptionNominalMax:    24800,
			ConsumptionLimit:                    0,
			FailsafeConsumptionActivePowerLimit: 4200,
			FailsafeDurationMinimum:             2 * time.Hour,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// get root circuit
	root := circuit.Root()
	if root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}

	// create new root circuit for LPC
	lpc, err := circuit.New(util.NewLogger("lpc"), "eebus", 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	// register LPC-Circuit for use in config, if not already registered
	if _, err := config.Circuits().ByName("lpc"); err != nil {
		_ = config.Circuits().Add(config.NewStaticDevice(config.Named{Name: "lpc"}, api.Circuit(lpc)))
	}

	// wrap old root with new pc parent
	if err := root.Wrap(lpc); err != nil {
		return nil, err
	}
	site.SetCircuit(lpc)

	return NewEEBus(ctx, cc.Ski, cc.Limits, lpc)
}

// NewEEBus creates EEBus charger
func NewEEBus(ctx context.Context, ski string, limits Limits, root api.Circuit) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:       util.NewLogger("eebus"),
		root:      root,
		uc:        eebus.Instance.ControllableSystem(),
		Connector: eebus.NewConnector(),
		heartbeat: util.NewValue[struct{}](2 * time.Minute), // LPC-031

		consumptionLimit: &ucapi.LoadLimit{
			Value:        limits.ConsumptionLimit,
			IsChangeable: true,
		},

		failsafeLimit:    limits.FailsafeConsumptionActivePowerLimit,
		failsafeDuration: limits.FailsafeDurationMinimum,
	}

	if err := eebus.Instance.RegisterDevice(ski, "", c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	// scenarios
	for _, s := range c.uc.LPC.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("LPC RemoteEntitiesScenarios:", s.Scenarios)
	}
	for _, s := range c.uc.LPP.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("LPP RemoteEntitiesScenarios:", s.Scenarios)
	}
	for _, s := range c.uc.MGCP.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("MGCP RemoteEntitiesScenarios:", s.Scenarios)
	}

	// set initial values
	if err := c.uc.LPC.SetConsumptionNominalMax(limits.ContractualConsumptionNominalMax); err != nil {
		c.log.ERROR.Println("LPC SetConsumptionNominalMax:", err)
	}
	if err := c.uc.LPC.SetConsumptionLimit(*c.consumptionLimit); err != nil {
		c.log.ERROR.Println("LPC SetConsumptionLimit:", err)
	}
	if err := c.uc.LPC.SetFailsafeConsumptionActivePowerLimit(c.failsafeLimit, true); err != nil {
		c.log.ERROR.Println("LPC SetFailsafeConsumptionActivePowerLimit:", err)
	}
	if err := c.uc.LPC.SetFailsafeDurationMinimum(c.failsafeDuration, true); err != nil {
		c.log.ERROR.Println("LPC SetFailsafeDurationMinimum:", err)
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
		if c.consumptionLimit != nil && c.consumptionLimit.IsActive {
			c.log.WARN.Println("active consumption limit")
			c.setStatusAndLimit(StatusLimited, c.consumptionLimit.Value)
		}

	case StatusLimited:
		// limit updated?
		if !c.consumptionLimit.IsActive {
			c.log.WARN.Println("inactive consumption limit")
			c.setStatusAndLimit(StatusUnlimited, 0)
			break
		}

		c.setLimit(c.consumptionLimit.Value)

		// LPC-914/1
		if d := c.consumptionLimit.Duration; d > 0 && time.Since(c.statusUpdated) > d {
			c.consumptionLimit = nil

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
