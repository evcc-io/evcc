package eebus

import (
	"context"
	"sync"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/config"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
)

func init() {
	config.AddCtx("eebus", NewFromConfig)
}

type EEBus struct {
	mux sync.RWMutex
	log *util.Logger

	*eebus.Connector
	cs *eebus.ControllableSystem

	site        site.API
	passthrough func(bool) error
	publishFunc func()

	status        status
	statusUpdated time.Time

	failsafeDuration time.Duration

	smartgridConsumptionId    uint
	consumptionLimit          ucapi.LoadLimit // LPC-041
	consumptionLimitActivated time.Time
	failsafeConsumptionLimit  float64

	smartgridProductionId    uint
	productionLimit          ucapi.LoadLimit // feed-in limit (NOT production despite its name)
	productionLimitActivated time.Time
	failsafeProductionLimit  *float64 // feed-in limit (NOT production despite its name)
	productionNominalMax     float64

	heartbeat *util.Value[struct{}]
	interval  time.Duration
}

type Limits struct {
	ContractualConsumptionNominalMax    float64
	FailsafeConsumptionActivePowerLimit float64

	ProductionNominalMax               float64
	FailsafeProductionActivePowerLimit *float64

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
			// contractual max power at the grid connection point reported to the control box
			// (EEBus LPC, EMS device type). Default: standard 3x35A x 230V house connection.
			// This is the connection capacity, not the SteuVE Pmin (see failsafe limit below).
			ContractualConsumptionNominalMax:    24150, // 3 * 35A * 230V
			FailsafeConsumptionActivePowerLimit: 4200,

			ProductionNominalMax:               0,
			FailsafeProductionActivePowerLimit: nil, // 0 is a valid limit

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

	return NewEEBus(ctx, cc.Ski, cc.Limits, passthroughS, site, cc.Interval)
}

// NewEEBus creates EEBus HEMS
func NewEEBus(ctx context.Context, ski string, limits Limits, passthrough func(bool) error, site site.API, interval time.Duration) (*EEBus, error) {
	inst, err := eebus.Instance()
	if err != nil {
		return nil, err
	}

	c := &EEBus{
		log:         util.NewLogger("eebus"),
		site:        site,
		passthrough: passthrough,
		cs:          inst.ControllableSystem(),
		Connector:   eebus.NewConnector(),
		heartbeat:   util.NewValue[struct{}](2 * time.Minute), // LPC-031
		interval:    interval,

		failsafeDuration:         limits.FailsafeDurationMinimum,
		failsafeConsumptionLimit: limits.FailsafeConsumptionActivePowerLimit,
		failsafeProductionLimit:  limits.FailsafeProductionActivePowerLimit,
		productionNominalMax:     limits.ProductionNominalMax,
	}

	// simulate a received heartbeat
	// otherwise a heartbeat timeout is assumed when the state machine is called for the first time
	c.heartbeat.Set(struct{}{})

	if err := inst.RegisterDevice(ski, "", c); err != nil {
		return nil, err
	}

	if err := c.Wait(ctx); err != nil {
		inst.UnregisterDevice(ski, c)
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
	if c.failsafeProductionLimit != nil && *c.failsafeProductionLimit >= 0 {
		if err := c.cs.CsLPPInterface.SetFailsafeProductionActivePowerLimit(*c.failsafeProductionLimit, true); err != nil {
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

func (c *EEBus) SetUpdated(f func()) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.publishFunc = f
}

func (c *EEBus) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}

		if c.publishFunc != nil {
			c.publishFunc()
		}
	}
}

func (c *EEBus) run() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("status:", c.status)

	_, heartbeatErr := c.heartbeat.Get()

	// LPC-911 / LPP-911: heartbeat lost while operating, enter failsafe.
	if heartbeatErr != nil && c.status != StatusFailsafe {
		c.log.WARN.Println("missing heartbeat- entering failsafe mode")
		c.setStatus(StatusFailsafe)

		c.setConsumptionLimit(c.failsafeConsumptionLimit)

		if c.failsafeProductionLimit != nil {
			// production limit is negative, failsafe limits are always positive
			c.setProductionLimit(-*c.failsafeProductionLimit, true)
		}

		return nil
	}

	if c.status == StatusFailsafe {
		if heartbeatErr != nil {
			// LPC-921 / LPP-921: still no heartbeat - keep applying the failsafe
			// limit. The failsafe limit is our self-determined protective default
			// for the Unlimited-autonomous state.
			return nil
		}

		// LPC-918/919/920 / LPP-equivalent: heartbeat returned - leave failsafe
		// immediately. Fall through to the LPC-914/1 block below, which will
		// apply whatever fresh limit the EG sent (or release the limit if the
		// EG has not sent an active limit since the failsafe entry).
		c.log.DEBUG.Println("heartbeat returned- leaving failsafe mode")
		c.setStatus(StatusNormal)

		c.setConsumptionLimit(0)
		c.setProductionLimit(0, false)
	}

	// LPC-914/1
	if c.consumptionLimitActivated.IsZero() {
		if c.consumptionLimit.IsActive {
			c.log.WARN.Println("activating consumption limit")
			c.setConsumptionLimit(c.consumptionLimit.Value)
		}
	} else {
		switch {
		case !c.consumptionLimit.IsActive:
			c.log.DEBUG.Println("consumption limit released")
			c.setConsumptionLimit(0)
		case time.Since(c.consumptionLimitActivated) > c.consumptionLimit.Duration:
			c.log.DEBUG.Println("consumption limit duration exceeded")
			c.setConsumptionLimit(0)
			c.consumptionLimit.IsActive = false
		}
	}

	// LPP
	if c.productionLimitActivated.IsZero() {
		if c.productionLimit.IsActive {
			c.log.WARN.Println("activating production limit")
			c.setProductionLimit(c.productionLimit.Value, true)
		}
	} else {
		switch {
		case !c.productionLimit.IsActive:
			c.log.DEBUG.Println("production limit released")
			c.setProductionLimit(0, false)
		case time.Since(c.productionLimitActivated) > c.productionLimit.Duration:
			c.log.DEBUG.Println("production limit duration exceeded")
			c.setProductionLimit(0, false)
			c.productionLimit.IsActive = false
		}
	}

	return nil
}

func (c *EEBus) setStatus(status status) {
	c.status = status
	c.statusUpdated = time.Now()
}

func (c *EEBus) setConsumptionLimit(limit float64) {
	active := limit > 0

	if active {
		c.consumptionLimitActivated = time.Now()
	} else {
		c.consumptionLimitActivated = time.Time{}
	}

	if err := smartgrid.UpdateSession(&c.smartgridConsumptionId, smartgrid.Dim, c.site.GetGridPower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	if c.passthrough != nil {
		if err := c.passthrough(limit > 0); err != nil {
			c.log.ERROR.Printf("passthrough failed: %v", err)
		}
	}
}

func (c *EEBus) setProductionLimit(limit float64, active bool) {
	if active {
		c.productionLimitActivated = time.Now()
	} else {
		c.productionLimitActivated = time.Time{}
	}

	if err := smartgrid.UpdateSession(&c.smartgridProductionId, smartgrid.Curtail, c.site.GetGridPower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}
}

var _ api.HEMS = (*EEBus)(nil)

// Dimmed implements api.HEMS, derived from consumptionLimitActivated.
func (c *EEBus) Dimmed() *bool {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return new(!c.consumptionLimitActivated.IsZero())
}

// CurtailedPercent implements api.HEMS, converting the active LPP production
// limit to an allowed production percent via the configured nominal production power.
func (c *EEBus) CurtailedPercent() *int {
	c.mux.RLock()
	defer c.mux.RUnlock()

	// without a nominal reference the W limit cannot be expressed as a percent
	if c.productionNominalMax <= 0 {
		return nil
	}

	percent := 100
	if !c.productionLimitActivated.IsZero() {
		// production limits are negative watts
		percent = int(-c.productionLimit.Value / c.productionNominalMax * 100)
	}

	return &percent
}

// MaxConsumptionPower implements api.HEMS, returning the consumption cap
// currently in effect: failsafe limit while in failsafe, otherwise the
// EG-supplied LPC limit when active, or 0 when no limit applies.
func (c *EEBus) MaxConsumptionPower() float64 {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.consumptionLimitActivated.IsZero() {
		return 0
	}
	if c.status == StatusFailsafe {
		return c.failsafeConsumptionLimit
	}
	return c.consumptionLimit.Value
}

// MaxProductionPower implements api.HEMS. Scaffolding only — EEBus does not
// publish a wattage-typed production cap yet.
func (c *EEBus) MaxProductionPower() *float64 {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.productionLimitActivated.IsZero() {
		return nil
	}
	if c.status == StatusFailsafe {
		return c.failsafeProductionLimit
	}
	return new(c.productionLimit.Value)
}
