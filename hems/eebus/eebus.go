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
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	cs *eebus.ControllableSystem

	root        api.Circuit
	passthrough func(bool) error

	status        status
	statusUpdated time.Time

	failsafeDuration time.Duration

	smartgridConsumptionId    uint
	consumptionLimit          ucapi.LoadLimit // LPC-041
	consumptionLimitActivated time.Time
	failsafeConsumptionLimit  float64

	smartgridProductionId    uint
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
		Ski         string
		Limits      `mapstructure:",squash"`
		Passthrough *plugin.Config
		Interval    time.Duration
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
	gridcontrol, err := shared.GetOrCreateCircuit("gridcontrol", "eebus")
	if err != nil {
		return nil, err
	}

	// wrap old root with new grid control parent
	if err := root.Wrap(gridcontrol); err != nil {
		return nil, err
	}
	site.SetCircuit(gridcontrol)

	return NewEEBus(ctx, cc.Ski, cc.Limits, passthroughS, gridcontrol, cc.Interval)
}

// NewEEBus creates EEBus HEMS
func NewEEBus(ctx context.Context, ski string, limits Limits, passthrough func(bool) error, root api.Circuit, interval time.Duration) (*EEBus, error) {
	if eebus.Instance == nil {
		return nil, errors.New("eebus not configured")
	}

	c := &EEBus{
		log:         util.NewLogger("eebus"),
		root:        root,
		passthrough: passthrough,
		cs:          eebus.Instance.ControllableSystem(),
		Connector:   eebus.NewConnector(),
		heartbeat:   util.NewValue[struct{}](2 * time.Minute), // LPC-031
		interval:    interval,

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
	eebus.LogEntities(c.log.DEBUG, "CS LPC", c.cs.CsLPCInterface)
	eebus.LogEntities(c.log.DEBUG, "CS LPP", c.cs.CsLPPInterface)

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

func (c *EEBus) run() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("status:", c.status)

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

	c.root.Dim(active)
	c.root.SetMaxPower(limit)

	if err := c.updateSession(&c.smartgridConsumptionId, smartgrid.Dim, limit); err != nil {
		c.log.ERROR.Printf("smartgrid dim session: %v", err)
	}

	if c.passthrough != nil {
		if err := c.passthrough(limit > 0); err != nil {
			c.log.ERROR.Printf("passthrough failed: %v", err)
		}
	}
}

func (c *EEBus) setProductionLimit(limit float64) {
	active := limit > 0

	if active {
		c.productionLimitActivated = time.Now()
	} else {
		c.productionLimitActivated = time.Time{}
	}

	c.root.Curtail(active)
	// TODO make ProductionNominalMax configurable (Site kWp)
	// c.root.SetMaxProduction(limit)

	if err := c.updateSession(&c.smartgridProductionId, smartgrid.Curtail, limit); err != nil {
		c.log.ERROR.Printf("smartgrid curtail session: %v", err)
	}
}

// TODO keep in sync across HEMS implementations
func (c *EEBus) updateSession(id *uint, typ smartgrid.Type, limit float64) error {
	// start session
	if limit > 0 && *id == 0 {
		var power *float64
		if p := c.root.GetChargePower(); p > 0 {
			power = new(p)
		}

		sid, err := smartgrid.StartManage(typ, power, limit)
		if err != nil {
			return err
		}

		*id = sid
	}

	// stop session
	if limit == 0 && *id != 0 {
		if err := smartgrid.StopManage(*id); err != nil {
			return err
		}

		*id = 0
	}

	return nil
}
