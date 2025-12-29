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

	lpc api.Circuit
	lpp api.Circuit

	status        status
	statusUpdated time.Time

	failsafeDuration time.Duration

	consumptionLimit          ucapi.LoadLimit // LPC-041
	consumptionLimitActivated time.Time
	failsafeConsumptionLimit  float64

	productionLimit          ucapi.LoadLimit
	productionLimitActivated time.Time
	failsafeProductionLimit  float64

	heartbeat *util.Value[struct{}]
	interval  time.Duration
}

type Limits struct {
	ContractualConsumptionNominalMax    float64
	FailsafeConsumptionActivePowerLimit float64

	ProductionNominalMax               float64
	FailsafeProductionActivePowerLimit float64

	FailsafeDurationMinimum time.Duration
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
			FailsafeConsumptionActivePowerLimit: 4200,

			ProductionNominalMax:               0,
			FailsafeProductionActivePowerLimit: 0,

			FailsafeDurationMinimum: 2 * time.Hour,
		},
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
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
	lpp, err := shared.GetOrCreateCircuit("lpp", "eebus")
	if err != nil {
		return nil, err
	}

	// wrap old root with new lpc parent
	if err := root.Wrap(lpc); err != nil {
		return nil, err
	}
	// wrap lpc with lpp parent
	if err := lpc.Wrap(lpp); err != nil {
		return nil, err
	}
	site.SetCircuit(lpp)

	return NewEEBus(ctx, cc.Ski, cc.Limits, lpc, lpp, cc.Interval)
}

// NewEEBus creates EEBus charger
func NewEEBus(ctx context.Context, ski string, limits Limits, lpc, lpp api.Circuit, interval time.Duration) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:       util.NewLogger("eebus"),
		lpc:       lpc,
		lpp:       lpp,
		cs:        eebus.Instance.ControllableSystem(),
		ma:        eebus.Instance.MonitoringAppliance(),
		eg:        eebus.Instance.EnergyGuard(),
		Connector: eebus.NewConnector(),
		heartbeat: util.NewValue[struct{}](2 * time.Minute), // LPC-031
		interval:  interval,

		failsafeDuration:         limits.FailsafeDurationMinimum,
		failsafeConsumptionLimit: limits.FailsafeConsumptionActivePowerLimit,
		failsafeProductionLimit:  limits.FailsafeProductionActivePowerLimit,
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
		c.log.DEBUG.Printf("ski %s CS LPC scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}
	for _, s := range c.cs.CsLPPInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Printf("ski %s CS LPP scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}

	// monitoring appliance
	for _, s := range c.ma.MaMPCInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Printf("ski %s MA MPC scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}
	for _, s := range c.ma.MaMGCPInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Printf("ski %s MA MGCP scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}

	// energy guard
	for _, s := range c.eg.EgLPCInterface.RemoteEntitiesScenarios() {
		c.log.DEBUG.Printf("ski %s EG LPC scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}

	// set initial values
	if err := c.cs.CsLPCInterface.SetConsumptionNominalMax(limits.ContractualConsumptionNominalMax); err != nil {
		c.log.ERROR.Println("CS LPC SetConsumptionNominalMax:", err)
	}
	if c.failsafeConsumptionLimit > 0 {
		if err := c.cs.CsLPCInterface.SetFailsafeConsumptionActivePowerLimit(c.failsafeConsumptionLimit, true); err != nil {
			c.log.ERROR.Println("CS LPC SetFailsafeConsumptionActivePowerLimit:", err)
		}
	}

	if err := c.cs.CsLPPInterface.SetProductionNominalMax(limits.ProductionNominalMax); err != nil {
		c.log.ERROR.Println("CS LPP SetProductionNominalMax:", err)
	}
	if c.failsafeProductionLimit > 0 {
		if err := c.cs.CsLPPInterface.SetFailsafeProductionActivePowerLimit(c.failsafeProductionLimit, true); err != nil {
			c.log.ERROR.Println("CS LPP SetFailsafeProductionActivePowerLimit:", err)
		}
	}

	if c.failsafeDuration > 0 {
		if err := c.cs.CsLPCInterface.SetFailsafeDurationMinimum(c.failsafeDuration, true); err != nil {
			c.log.ERROR.Println("CS LPC SetFailsafeDurationMinimum:", err)
		}
		if err := c.cs.CsLPPInterface.SetFailsafeDurationMinimum(c.failsafeDuration, true); err != nil {
			c.log.ERROR.Println("CS LPP SetFailsafeDurationMinimum:", err)
		}
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

func (c *EEBus) run() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("status: ", c.status)

	// check heartbeat
	_, heartbeatErr := c.heartbeat.Get()
	if heartbeatErr != nil && c.status != StatusFailsafe {
		// LPC-914/2
		c.log.WARN.Println("missing heartbeat- entering failsafe mode")
		c.setStatusAndLimit(StatusFailsafe, c.failsafeConsumptionLimit, c.failsafeProductionLimit)

		return nil
	}

	if c.status == StatusFailsafe {
		// LPC-914/2
		if heartbeatErr != nil || time.Since(c.statusUpdated) <= c.failsafeDuration {
			return nil
		}

		c.log.DEBUG.Println("heartbeat returned or failsafe duration exceeded- leaving failsafe mode")
		c.setStatusAndLimit(StatusNormal, 0, 0)
	}

	// LPC-914/1
	if c.consumptionLimitActivated.IsZero() {
		if c.consumptionLimit.IsActive {
			c.log.WARN.Println("activating consumption limit")
			c.setConsumptionLimit(c.consumptionLimit.Value)
		}
	} else {
		if time.Since(c.consumptionLimitActivated) > c.consumptionLimit.Duration {
			c.log.DEBUG.Println("consumption limit duration exceeded")
			c.setConsumptionLimit(0)
		}
	}

	// LPP
	if c.productionLimitActivated.IsZero() {
		if c.productionLimit.IsActive {
			c.log.WARN.Println("activating production limit")
			c.setProductionLimit(c.productionLimit.Value)
		}
	} else {
		if time.Since(c.productionLimitActivated) > c.productionLimit.Duration {
			c.log.DEBUG.Println("production limit duration exceeded")
			c.setProductionLimit(0)
		}
	}

	return nil
}

func (c *EEBus) setStatusAndLimit(status status, consumption, production float64) {
	c.status = status
	c.statusUpdated = time.Now()

	c.setConsumptionLimit(consumption)
	c.setProductionLimit(production)
}

func (c *EEBus) setConsumptionLimit(limit float64) {
	active := limit > 0

	if active {
		c.consumptionLimitActivated = time.Now()
	} else {
		c.consumptionLimitActivated = time.Time{}
	}

	c.lpc.Dim(active)
	c.lpc.SetMaxPower(limit)
}

func (c *EEBus) setProductionLimit(limit float64) {
	active := limit > 0

	if active {
		c.productionLimitActivated = time.Now()
	} else {
		c.productionLimitActivated = time.Time{}
	}

	// TODO curtail
	// c.lpp.Dim(active)
	// c.lpp.SetMaxPower(limit)
}
