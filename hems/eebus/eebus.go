package eebus

import (
	"context"
	"errors"
	"sync"
	"time"

	//eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/shared"
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

	lpc_root api.Circuit
	lpp_root api.Circuit

	status         status
	CstatusUpdated time.Time
	PstatusUpdated time.Time

	consumptionLimit *ucapi.LoadLimit // LPC-041
	productionLimit  *ucapi.LoadLimit

	failsafeLimit              float64
	failsafeDuration           time.Duration
	productionfailsafeLimit    float64
	productionfailsafeDuration time.Duration

	heartbeat *util.Value[struct{}]
	interval  time.Duration
}

type Limits struct {
	ContractualConsumptionNominalMax    float64
	ConsumptionLimit                    float64
	FailsafeConsumptionActivePowerLimit float64
	FailsafeDurationMinimum             time.Duration
	ProductionNominalMax                float64
	ProductionLimit                     float64
	FailsafeProductionActivePowerLimit  float64
	ProductionFailsafeDurationMinimum   time.Duration
}

// NewFromConfig creates an EEBus HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*EEBus, error) {
	cc := struct {
		Ski      string
		Limits   `mapstructure:",squash"`
		Interval time.Duration
	}{
		Limits: Limits{
			ContractualConsumptionNominalMax:    24800,
			ConsumptionLimit:                    0,
			FailsafeConsumptionActivePowerLimit: 4200,
			FailsafeDurationMinimum:             2 * time.Hour,
			ProductionNominalMax:                24800, // e.g. for bidirectional chargers, home batteries, pv inverters
			ProductionLimit:                     0,
			FailsafeProductionActivePowerLimit:  4200,
			ProductionFailsafeDurationMinimum:   2 * time.Hour,
		},
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// get root circuit
	lpc_root := circuit.Root()
	if lpc_root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}

	//todo: root circuit for lpp

	// register LPC circuit if not already registered
	lpc, err := shared.GetOrCreateCircuit("lpc", "eebus")
	if err != nil {
		return nil, err
	}
	lpp, err := shared.GetOrCreateCircuit("lpp", "eebus")
	if err != nil {
		return nil, err
	}

	// wrap old root with new pc parent
	if err := lpc_root.Wrap(lpc); err != nil {
		return nil, err
	}
	site.SetCircuit(lpc)
	site.SetCircuit(lpp)

	return NewEEBus(ctx, cc.Ski, cc.Limits, lpc, lpp, cc.Interval)
}

//LPP

// NewEEBus creates EEBus charger
func NewEEBus(ctx context.Context, ski string, limits Limits, lpc_root api.Circuit, lpp_root api.Circuit, interval time.Duration) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:       util.NewLogger("eebus"),
		lpc_root:  lpc_root,
		lpp_root:  lpp_root,
		cs:        eebus.Instance.ControllableSystem(),
		ma:        eebus.Instance.MonitoringAppliance(),
		eg:        eebus.Instance.EnergyGuard(),
		Connector: eebus.NewConnector(),
		heartbeat: util.NewValue[struct{}](2 * time.Minute), // LPC-031
		interval:  interval,

		consumptionLimit: &ucapi.LoadLimit{
			Value:        limits.ConsumptionLimit,
			IsChangeable: true,
		},

		failsafeLimit:    limits.FailsafeConsumptionActivePowerLimit,
		failsafeDuration: limits.FailsafeDurationMinimum,

		productionLimit: &ucapi.LoadLimit{
			Value:        limits.ProductionLimit,
			IsChangeable: true,
		},

		productionfailsafeLimit:    limits.FailsafeProductionActivePowerLimit,
		productionfailsafeDuration: limits.ProductionFailsafeDurationMinimum,
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
	for t, s := range c.cs.CsLPCInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Println("CS LPC RemoteEntitiesScenarios:", t, s.Scenarios)
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

	if err := c.cs.CsLPPInterface.SetProductionNominalMax(limits.ProductionNominalMax); err != nil {
		c.log.ERROR.Println("CS LPP SetProductionNominalMax:", err)
	}
	if err := c.cs.CsLPPInterface.SetProductionLimit(*c.productionLimit); err != nil {
		c.log.ERROR.Println("CS LPP SetProductionLimit:", err)
	}
	if err := c.cs.CsLPPInterface.SetFailsafeProductionActivePowerLimit(c.productionfailsafeLimit, true); err != nil {
		c.log.ERROR.Println("CS LPP SetFailsafeProductionActivePowerLimit:", err)
	}
	if err := c.cs.CsLPPInterface.SetFailsafeDurationMinimum(c.productionfailsafeDuration, true); err != nil {
		c.log.ERROR.Println("CS LPP SetFailsafeDurationMinimum:", err)
	}

	return c, nil
}

func (c *EEBus) Run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
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
		c.setStatusAndLimit(StatusFailsafe, c.failsafeLimit, true)
		c.setLPPStatusAndLimit(StatusFailsafe, c.productionfailsafeLimit, true)

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
			c.setStatusAndLimit(StatusLimited, c.consumptionLimit.Value, true)
		}

		if c.productionLimit != nil && c.productionLimit.IsActive {
			c.log.WARN.Println("active production limit")
			// production limit is negative, but I want positive
			c.setLPPStatusAndLimit(StatusLimited, -1*c.productionLimit.Value, true)
		}

	case StatusLimited:
		// limit updated?
		if !c.consumptionLimit.IsActive && !c.productionLimit.IsActive {
			c.log.WARN.Println("inactive consumption limit, inactive production limit")
			c.setStatusAndLimit(StatusUnlimited, 0, false)
			c.setLPPStatusAndLimit(StatusUnlimited, 0, false)
			// break
		} else if !c.consumptionLimit.IsActive && c.productionLimit.IsActive {
			c.log.WARN.Println("inactive consumption limit, active production limit")
			c.setStatusAndLimit(StatusLimited, c.consumptionLimit.Value, false)
			c.setLPPLimit(-1*c.productionLimit.Value, true)
		} else if c.consumptionLimit.IsActive && !c.productionLimit.IsActive {
			c.log.WARN.Println("active consumption limit, inactive production limit")
			c.setLimit(c.consumptionLimit.Value, true)
			c.setLPPStatusAndLimit(StatusLimited, 0, false)
		} else if c.consumptionLimit.IsActive && c.productionLimit.IsActive {
			// both limits active - senceless, but possible
			c.log.WARN.Println("active consumption limit, active production limit")
			c.setLimit(c.consumptionLimit.Value, true)
			c.setLPPLimit(-1*c.productionLimit.Value, true)
		}

		// LPC-914/1
		if d := c.consumptionLimit.Duration; d > 0 && time.Since(c.CstatusUpdated) > d {
			c.consumptionLimit.IsActive = false

			c.log.DEBUG.Println("consumption limit duration exceeded- return to normal")
			c.setLimit(0, false)
		}

		if d := c.productionLimit.Duration; d > 0 && time.Since(c.PstatusUpdated) > d {
			c.productionLimit.IsActive = false

			c.log.DEBUG.Println("production limit duration exceeded- return to normal")
			c.setLPPLimit(0, false)
		}

	case StatusFailsafe:
		// LPC-914/2
		now := time.Now()
		cExceeded := now.Sub(c.CstatusUpdated) > c.failsafeDuration
		pExceeded := now.Sub(c.PstatusUpdated) > c.productionfailsafeDuration

		if cExceeded {
			c.log.DEBUG.Println("Consumption failsafe duration exceeded- returned to normal")
			c.setLimit(0, false)
		}

		if pExceeded {
			c.log.DEBUG.Println("Production failsafe duration exceeded- returned to normal")
			c.setLPPLimit(0, false)
		}

		if heartbeatErr == nil {
			c.log.DEBUG.Println("heartbeat returned leaving failsafe mode")
			c.setStatusAndLimit(StatusUnlimited, 0, false)
			c.setLPPStatusAndLimit(StatusUnlimited, 0, false)
		}
	}

	return nil
}

func (c *EEBus) setStatusAndLimit(status status, limit float64, dimmed bool) {
	c.status = status
	c.CstatusUpdated = time.Now()

	c.setLimit(limit, dimmed)
}

func (c *EEBus) setLimit(limit float64, dimmed bool) {
	c.lpc_root.Dim(dimmed)
	c.lpc_root.SetMaxPower(limit)
}
func (c *EEBus) setLPPStatusAndLimit(status status, limit float64, dimmed bool) {
	c.status = status
	c.PstatusUpdated = time.Now()

	c.setLPPLimit(limit, dimmed)
}

func (c *EEBus) setLPPLimit(limit float64, dimmed bool) {
	c.lpp_root.Dim(dimmed)
	c.lpp_root.SetMaxPower(limit)
}
