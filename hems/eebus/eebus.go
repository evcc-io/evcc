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
	"github.com/evcc-io/evcc/hems/shared"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	cs *eebus.ControllableSystem
	ma *eebus.MonitoringAppliance
	eg *eebus.EnergyGuard

	root        api.Circuit
	passthrough func(bool) error

	status        status
	statusUpdated time.Time

	consumptionLimit *ucapi.LoadLimit // LPC-041
	failsafeLimit    float64
	failsafeDuration time.Duration

	heartbeat *util.Value[struct{}]
	interval  time.Duration
}

type Limits struct {
	ContractualConsumptionNominalMax    float64
	ConsumptionLimit                    float64
	FailsafeConsumptionActivePowerLimit float64
	FailsafeDurationMinimum             time.Duration
}

// NewFromConfig creates an EEBus HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*EEBus, error) {
	cc := struct {
		Ski         string
		Limits      `mapstructure:",squash"`
		Passthrough *plugin.Config
		Interval    time.Duration
	}{
		Limits: Limits{
			ContractualConsumptionNominalMax:    24800,
			ConsumptionLimit:                    0,
			FailsafeConsumptionActivePowerLimit: 4200,
			FailsafeDurationMinimum:             2 * time.Hour,
		},
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	passthroughS, err := cc.Passthrough.BoolSetter(ctx, "dim")
	if err != nil {
		return nil, err
	}

	// get root circuit
	root := circuit.Root()
	if root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}

	// register LPC circuit if not already registered
	lpc, err := shared.GetOrCreateCircuit("lpc", "eebus")
	if err != nil {
		return nil, err
	}

	// wrap old root with new pc parent
	if err := root.Wrap(lpc); err != nil {
		return nil, err
	}
	site.SetCircuit(lpc)

	return NewEEBus(ctx, cc.Ski, cc.Limits, passthroughS, lpc, cc.Interval)
}

// NewEEBus creates EEBus charger
func NewEEBus(ctx context.Context, ski string, limits Limits, passthrough func(bool) error, root api.Circuit, interval time.Duration) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:         util.NewLogger("eebus"),
		root:        root,
		passthrough: passthrough,
		cs:          eebus.Instance.ControllableSystem(),
		ma:          eebus.Instance.MonitoringAppliance(),
		eg:          eebus.Instance.EnergyGuard(),
		Connector:   eebus.NewConnector(),
		heartbeat:   util.NewValue[struct{}](2 * time.Minute), // LPC-031
		interval:    interval,

		consumptionLimit: &ucapi.LoadLimit{
			Value:        limits.ConsumptionLimit,
			IsChangeable: true,
		},

		failsafeLimit:    limits.FailsafeConsumptionActivePowerLimit,
		failsafeDuration: limits.FailsafeDurationMinimum,
	}

	// simulate a received heartbeat
	// otherwise a heartbeat timeout is assumed when the state machine is called for the first time
	c.heartbeat.Set(struct{}{})

	if err := eebus.Instance.RegisterDevice(ski, "", c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	// controllable system
	for _, s := range c.cs.CsLPCInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("CS LPC RemoteEntitiesScenarios:", s.Scenarios)
	}
	for _, s := range c.cs.CsLPPInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("CS LPP RemoteEntitiesScenarios:", s.Scenarios)
	}

	// monitoring appliance
	for _, s := range c.ma.MaMPCInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("MA MPC RemoteEntitiesScenarios:", s.Scenarios)
	}
	for _, s := range c.ma.MaMGCPInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("MA MGCP RemoteEntitiesScenarios:", s.Scenarios)
	}

	// energy guard
	for _, s := range c.eg.EgLPCInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Println("EG LPC RemoteEntitiesScenarios:", s.Scenarios)
	}

	// set initial values
	if err := c.cs.CsLPCInterface.SetConsumptionNominalMax(limits.ContractualConsumptionNominalMax); err != nil {
		c.log.ERROR.Println("CS LPC SetConsumptionNominalMax:", err)
	}
	if err := c.cs.CsLPCInterface.SetConsumptionLimit(*c.consumptionLimit); err != nil {
		c.log.ERROR.Println("CS LPC SetConsumptionLimit:", err)
	}
	if err := c.cs.CsLPCInterface.SetFailsafeConsumptionActivePowerLimit(c.failsafeLimit, true); err != nil {
		c.log.ERROR.Println("CS LPC SetFailsafeConsumptionActivePowerLimit:", err)
	}
	if err := c.cs.CsLPCInterface.SetFailsafeDurationMinimum(c.failsafeDuration, true); err != nil {
		c.log.ERROR.Println("CS LPC SetFailsafeDurationMinimum:", err)
	}

	return c, nil
}

func (c *EEBus) ConsumptionLimit() float64 {
	return c.consumptionLimit.Value
}

func (c *EEBus) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

// TODO check state machine against spec
func (c *EEBus) run() error {
	c.mux.Lock()
	defer c.mux.Unlock()

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
		if d := c.failsafeDuration; heartbeatErr == nil || time.Since(c.statusUpdated) > d {
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
	c.root.Dim(limit > 0)
	c.root.SetMaxPower(limit)

	if c.passthrough != nil {
		if err := c.passthrough(limit > 0); err != nil {
			c.log.ERROR.Printf("passthrough failed: %v", err)
		}
	}
}
