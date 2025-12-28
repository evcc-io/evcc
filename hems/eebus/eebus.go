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
			ProductionNominalMax:                0,
			ProductionLimit:                     0,
			FailsafeProductionActivePowerLimit:  0,
			FailsafeProductionDurationMinimum:   2 * time.Hour,
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

		consumptionLimit: &ucapi.LoadLimit{
			Value:        limits.ConsumptionLimit,
			IsChangeable: true,
		},

		failsafeConsumptionLimit:    limits.FailsafeConsumptionActivePowerLimit,
		failsafeConsumptionDuration: limits.FailsafeConsumptionDurationMinimum,

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
	if err := c.cs.CsLPCInterface.SetFailsafeDurationMinimum(c.failsafeConsumptionDuration, true); err != nil {
		c.log.ERROR.Println("CS LPC SetFailsafeDurationMinimum:", err)
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
	if err := c.cs.CsLPPInterface.SetFailsafeDurationMinimum(c.failsafeProductionDuration, true); err != nil {
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

	c.log.TRACE.Println("consumption status:", c.consumptionStatus)
	c.log.TRACE.Println("production status:", c.productionStatus)

	// check heartbeat
	_, heartbeatErr := c.consumptionHeartbeat.Get()
	if heartbeatErr != nil && c.consumptionStatus != StatusFailsafe {
		// LPC-914/2

		// TODO fix status handling
		c.log.WARN.Println("missing consumption heartbeat- entering failsafe mode")
		c.setLPCStatusAndLimit(StatusFailsafe, c.failsafeConsumptionLimit, true)
		c.setLPPStatusAndLimit(StatusFailsafe, c.failsafeProductionLimit, true)

		return nil
	}

	// TODO
	// status init
	// status Unlimited/controlled
	// status Unlimited/autonomous

	switch c.consumptionStatus {
	case StatusUnlimited:
		// LPC-914/1
		if c.consumptionLimit != nil && c.consumptionLimit.IsActive {
			c.log.WARN.Println("active consumption limit")
			c.setLPCStatusAndLimit(StatusLimited, c.consumptionLimit.Value, true)
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
			c.setLPCStatusAndLimit(StatusUnlimited, 0, false)
			c.setLPPStatusAndLimit(StatusUnlimited, 0, false)
			// break
		} else if !c.consumptionLimit.IsActive && c.productionLimit.IsActive {
			c.log.WARN.Println("inactive consumption limit, active production limit")
			c.setLPCStatusAndLimit(StatusLimited, c.consumptionLimit.Value, false)
			c.setLPPLimit(-1*c.productionLimit.Value, true)
		} else if c.consumptionLimit.IsActive && !c.productionLimit.IsActive {
			c.log.WARN.Println("active consumption limit, inactive production limit")
			c.setLPCLimit(c.consumptionLimit.Value, true)
			c.setLPPStatusAndLimit(StatusLimited, 0, false)
		} else if c.consumptionLimit.IsActive && c.productionLimit.IsActive {
			// both limits active - senceless, but possible
			c.log.WARN.Println("active consumption limit, active production limit")
			c.setLPCLimit(c.consumptionLimit.Value, true)
			c.setLPPLimit(-1*c.productionLimit.Value, true)
		}

		// LPC-914/1
		if d := c.consumptionLimit.Duration; d > 0 && time.Since(c.consumptionStatusUpdated) > d {
			c.consumptionLimit.IsActive = false

			c.log.DEBUG.Println("consumption limit duration exceeded- return to normal")
			c.setLPCLimit(0, false)
		}

		if d := c.productionLimit.Duration; d > 0 && time.Since(c.productionStatusUpdated) > d {
			c.productionLimit.IsActive = false

			c.log.DEBUG.Println("production limit duration exceeded- return to normal")
			c.setLPPLimit(0, false)
		}

	case StatusFailsafe:
		// LPC-914/2
		now := time.Now()
		cExceeded := now.Sub(c.consumptionStatusUpdated) > c.failsafeConsumptionDuration
		pExceeded := now.Sub(c.productionStatusUpdated) > c.failsafeProductionDuration

		if cExceeded {
			c.log.DEBUG.Println("Consumption failsafe duration exceeded- returned to normal")
			c.setLPCLimit(0, false)
		}

		if pExceeded {
			c.log.DEBUG.Println("Production failsafe duration exceeded- returned to normal")
			c.setLPPLimit(0, false)
		}

		if heartbeatErr == nil {
			c.log.DEBUG.Println("heartbeat returned leaving failsafe mode")
			c.setLPCStatusAndLimit(StatusUnlimited, 0, false)
			c.setLPPStatusAndLimit(StatusUnlimited, 0, false)
		}
	}

	return nil
}

func (c *EEBus) setLPCStatusAndLimit(status status, limit float64, dimmed bool) {
	c.consumptionStatus = status
	c.consumptionStatusUpdated = time.Now()

	c.setLPCLimit(limit, dimmed)
}

func (c *EEBus) setLPCLimit(limit float64, dimmed bool) {
	c.lpc.Dim(dimmed)
	c.lpc.SetMaxPower(limit)
}

func (c *EEBus) setLPPStatusAndLimit(status status, limit float64, dimmed bool) {
	c.productionStatus = status
	c.productionStatusUpdated = time.Now()

	c.setLPPLimit(limit, dimmed)
}

func (c *EEBus) setLPPLimit(limit float64, dimmed bool) {
	// TODO curtail
	c.lpp.Dim(dimmed)
	c.lpp.SetMaxPower(limit)
}
