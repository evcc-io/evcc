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

//go:generate go run gen/main.go consumption production
type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	cs *eebus.ControllableSystem
	ma *eebus.MonitoringAppliance
	eg *eebus.EnergyGuard

	lpc api.Circuit
	lpp api.Circuit

	consumptionStatus        status
	consumptionStatusUpdated time.Time
	productionStatus         status
	productionStatusUpdated  time.Time

	consumptionHeartbeat        *util.Value[struct{}]
	consumptionLimit            *ucapi.LoadLimit // LPC-041
	failsafeConsumptionLimit    float64
	failsafeConsumptionDuration time.Duration

	productionHeartbeat        *util.Value[struct{}]
	productionLimit            *ucapi.LoadLimit
	failsafeProductionLimit    float64
	failsafeProductionDuration time.Duration

	interval time.Duration
}

type Limits struct {
	ContractualConsumptionNominalMax    float64
	ConsumptionLimit                    float64
	FailsafeConsumptionActivePowerLimit float64
	FailsafeConsumptionDurationMinimum  time.Duration

	ProductionNominalMax               float64
	ProductionLimit                    float64
	FailsafeProductionActivePowerLimit float64
	FailsafeProductionDurationMinimum  time.Duration
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
			FailsafeConsumptionDurationMinimum:  2 * time.Hour,

			ProductionNominalMax:               0,
			ProductionLimit:                    0,
			FailsafeProductionActivePowerLimit: 0,
			FailsafeProductionDurationMinimum:  2 * time.Hour,
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

//LPP

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
		interval:  interval,

		consumptionHeartbeat: util.NewValue[struct{}](2 * time.Minute), // LPC-031
		consumptionLimit: &ucapi.LoadLimit{
			Value:        limits.ConsumptionLimit,
			IsChangeable: true,
		},

		failsafeConsumptionLimit:    limits.FailsafeConsumptionActivePowerLimit,
		failsafeConsumptionDuration: limits.FailsafeConsumptionDurationMinimum,

		productionHeartbeat: util.NewValue[struct{}](2 * time.Minute), // LPC-031
		productionLimit: &ucapi.LoadLimit{
			Value:        limits.ProductionLimit,
			IsChangeable: true,
		},

		failsafeProductionLimit:    limits.FailsafeProductionActivePowerLimit,
		failsafeProductionDuration: limits.FailsafeProductionDurationMinimum,
	}

	// simulate a received heartbeat
	// otherwise a heartbeat timeout is assumed when the state machine is called for the first time
	c.consumptionHeartbeat.Set(struct{}{})
	c.productionHeartbeat.Set(struct{}{})

	if err := eebus.Instance.RegisterDevice(ski, "", c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
		eebus.Instance.UnregisterDevice(ski, c)
		return nil, err
	}

	// controllable system
	for _, s := range c.cs.CsLPCInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Printf("ski %s CS LPC scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}
	for _, s := range c.cs.CsLPPInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Printf("ski %s CS LPP scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}

	// monitoring appliance
	for _, s := range c.ma.MaMPCInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Printf("ski %s MA MPC scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}
	for _, s := range c.ma.MaMGCPInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Printf("ski %s MA MGCP scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}

	// energy guard
	for _, s := range c.eg.EgLPCInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Printf("ski %s EG LPC scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}
	for _, s := range c.eg.EgLPPInterface.RemoteEntitiesScenarios() {
		c.log.INFO.Printf("ski %s EG LPP scenarios: %v", s.Entity.Device().Ski(), s.Scenarios)
	}

	// set initial values
	if err := c.cs.CsLPCInterface.SetConsumptionNominalMax(limits.ContractualConsumptionNominalMax); err != nil {
		c.log.ERROR.Println("CS LPC SetConsumptionNominalMax:", err)
	}
	if c.consumptionLimit.Value > 0 {
		if err := c.cs.CsLPCInterface.SetConsumptionLimit(*c.consumptionLimit); err != nil {
			c.log.ERROR.Println("CS LPC SetConsumptionLimit:", err)
		}
	}
	if c.failsafeConsumptionLimit > 0 {
		if err := c.cs.CsLPCInterface.SetFailsafeConsumptionActivePowerLimit(c.failsafeConsumptionLimit, true); err != nil {
			c.log.ERROR.Println("CS LPC SetFailsafeConsumptionActivePowerLimit:", err)
		}
	}
	if c.failsafeConsumptionDuration > 0 {
		if err := c.cs.CsLPCInterface.SetFailsafeDurationMinimum(c.failsafeConsumptionDuration, true); err != nil {
			c.log.ERROR.Println("CS LPC SetFailsafeDurationMinimum:", err)
		}
	}

	if err := c.cs.CsLPPInterface.SetProductionNominalMax(limits.ProductionNominalMax); err != nil {
		c.log.ERROR.Println("CS LPP SetProductionNominalMax:", err)
	}
	if c.productionLimit.Value > 0 {
		if err := c.cs.CsLPPInterface.SetProductionLimit(*c.productionLimit); err != nil {
			c.log.ERROR.Println("CS LPP SetProductionLimit:", err)
		}
	}
	if c.failsafeProductionLimit > 0 {
		if err := c.cs.CsLPPInterface.SetFailsafeProductionActivePowerLimit(c.failsafeProductionLimit, true); err != nil {
			c.log.ERROR.Println("CS LPP SetFailsafeProductionActivePowerLimit:", err)
		}
	}
	if c.failsafeProductionDuration > 0 {
		if err := c.cs.CsLPPInterface.SetFailsafeDurationMinimum(c.failsafeProductionDuration, true); err != nil {
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

	return errors.Join(
		c.handleConsumption(),
		c.handleProduction(),
	)
}

func (c *EEBus) setConsumptionLimit(limit float64) {
	c.lpc.Dim(limit > 0)
	c.lpc.SetMaxPower(limit)
}

func (c *EEBus) setProductionLimit(limit float64) {
	// TODO curtail
	// c.lpp.Dim(limit > 0)
	// c.lpp.SetMaxPower(limit)
}
