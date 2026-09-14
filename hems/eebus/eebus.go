package eebus

import (
	"context"
	"errors"
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
	ctx context.Context // device lifetime, aborts Run
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
	consumptionLimitActivated *time.Time      // nil until first connected, then always set
	failsafeConsumptionLimit  float64

	smartgridProductionId    uint
	productionLimit          ucapi.LoadLimit // feed-in limit (NOT production despite its name)
	productionLimitActivated *time.Time      // nil until first connected, then always set
	failsafeProductionLimit  *float64        // feed-in limit (NOT production despite its name)
	productionNominalMax     float64

	heartbeat         *util.Value[struct{}]
	heartbeatReturned time.Time // heartbeat resumed while in failsafe
	limitReceived     time.Time // last limit written by the Energy Guard
	interval          time.Duration
}

// failsafeReleaseTimeout is how long the CS keeps the failsafe limit after the
// heartbeat resumed but the Energy Guard has not stated a limit yet ([LPC-921]).
const failsafeReleaseTimeout = 2 * time.Minute

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
		ctx:         ctx,
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
	if limits.ContractualConsumptionNominalMax > 0 {
		if err := c.cs.CsLPCInterface.SetConsumptionNominalMax(limits.ContractualConsumptionNominalMax); err != nil {
			c.log.ERROR.Println("CS LPC SetConsumptionNominalMax:", err)
		}
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

// Connect overrides the embedded Connector: on first connect, limit state
// becomes valid (nil -> known). A later disconnect/reconnect is a no-op here.
func (c *EEBus) Connect(connected bool) {
	c.Connector.Connect(connected)

	if !connected {
		return
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	if c.consumptionLimitActivated == nil {
		c.consumptionLimitActivated = new(time.Time)
	}
	if c.productionLimitActivated == nil {
		c.productionLimitActivated = new(time.Time)
	}
}

// Run applies limits until the device context is cancelled
func (c *EEBus) Run() {
	// LPC-TS-017: the first run applies the failsafe limit until the Energy Guard states one
	for tick := time.Tick(c.interval); ; {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}

		if c.publishFunc != nil {
			c.publishFunc()
		}

		select {
		case <-tick:
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *EEBus) run() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.TRACE.Println("status:", c.status)

	_, heartbeatErr := c.heartbeat.Get()

	// the LPC-921 release window only runs while the heartbeat is back
	if heartbeatErr != nil {
		c.heartbeatReturned = time.Time{}
	}

	// LPC-911 / LPP-911: heartbeat lost while operating, enter failsafe.
	if heartbeatErr != nil && c.status != StatusFailsafe {
		c.log.WARN.Println("missing heartbeat- entering failsafe mode")
		c.setStatus(StatusFailsafe)

		c.setConsumptionLimit(c.failsafeConsumptionLimit)

		if c.failsafeProductionLimit != nil {
			// production limit is negative, failsafe limits are always positive
			c.setProductionLimit(-*c.failsafeProductionLimit, true)
		} else {
			// no failsafe limit configured, release any stale Energy Guard limit
			c.setProductionLimit(0, false)
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

		if c.heartbeatReturned.IsZero() {
			c.heartbeatReturned = time.Now()
		}

		// LPC-916/LPP-916: the failsafe state is left on a heartbeat and a *following*
		// limit write. Without one, LPC-921 grants 120s before going unlimited.
		if c.limitReceived.Before(c.statusUpdated) && time.Since(c.heartbeatReturned) < failsafeReleaseTimeout {
			return nil
		}

		// LPC-918/919/920 / LPP-equivalent: leave failsafe. Fall through to the
		// LPC-914/1 block below, which will apply whatever fresh limit the EG sent
		// (or release the limit if the EG has not sent an active limit).
		c.log.DEBUG.Println("heartbeat returned- leaving failsafe mode")
		c.setStatus(StatusNormal)

		c.setConsumptionLimit(0)
		c.setProductionLimit(0, false)
	}

	// LPC-914/1
	if !limitActive(c.consumptionLimitActivated) {
		if c.consumptionLimit.IsActive {
			c.log.WARN.Println("activating consumption limit")
			c.setConsumptionLimit(c.consumptionLimit.Value)
		}
	} else {
		switch {
		case !c.consumptionLimit.IsActive:
			c.log.DEBUG.Println("consumption limit released")
			c.setConsumptionLimit(0)
		// a limit stated without duration does not expire
		case c.consumptionLimit.Duration > 0 && time.Since(*c.consumptionLimitActivated) > c.consumptionLimit.Duration:
			c.log.DEBUG.Println("consumption limit duration exceeded")
			c.setConsumptionLimit(0)
			c.consumptionLimit.IsActive = false
		}
	}

	// LPP
	if !limitActive(c.productionLimitActivated) {
		if c.productionLimit.IsActive {
			if c.productionNominalMax <= 0 {
				return errors.New("production limit received but productionNominalMax is not configured")
			}

			c.log.WARN.Println("activating production limit")
			c.setProductionLimit(c.productionLimit.Value, true)
		}
	} else {
		switch {
		case !c.productionLimit.IsActive:
			c.log.DEBUG.Println("production limit released")
			c.setProductionLimit(0, false)
		// a limit stated without duration does not expire
		case c.productionLimit.Duration > 0 && time.Since(*c.productionLimitActivated) > c.productionLimit.Duration:
			c.log.DEBUG.Println("production limit duration exceeded")
			c.setProductionLimit(0, false)
			c.productionLimit.IsActive = false
		}
	}

	return nil
}

// limitActive reports whether t denotes a currently active limit: known (non-nil) and non-zero.
func limitActive(t *time.Time) bool {
	return t != nil && !t.IsZero()
}

// activatedAt returns now if active, else a known-but-zero timestamp.
func activatedAt(active bool) *time.Time {
	if active {
		t := time.Now()
		return &t
	}
	return new(time.Time)
}

func (c *EEBus) setStatus(status status) {
	c.status = status
	c.statusUpdated = time.Now()
}

func (c *EEBus) setConsumptionLimit(limit float64) {
	active := limit > 0
	c.consumptionLimitActivated = activatedAt(active)

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
	c.productionLimitActivated = activatedAt(active)

	if err := smartgrid.UpdateSession(&c.smartgridProductionId, smartgrid.Curtail, c.site.GetGridPower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}
}

// effectiveConsumptionLimit returns the LPC limit in force: the configured failsafe
// limit while in failsafe, else the Energy Guard's. Caller holds the mutex.
func (c *EEBus) effectiveConsumptionLimit() float64 {
	if c.status == StatusFailsafe {
		return c.failsafeConsumptionLimit
	}
	return c.consumptionLimit.Value
}

// effectiveProductionLimit returns the LPP limit in force as positive watts, see
// effectiveConsumptionLimit. LPP states its limits as negative watts.
func (c *EEBus) effectiveProductionLimit() float64 {
	if c.status == StatusFailsafe && c.failsafeProductionLimit != nil {
		return *c.failsafeProductionLimit
	}
	return -c.productionLimit.Value
}

var _ api.HEMS = (*EEBus)(nil)

// CurtailedPercent implements api.HEMS, converting the active LPP production
// limit to an allowed production percent via the configured nominal production power.
func (c *EEBus) CurtailedPercent() *int {
	c.mux.RLock()
	defer c.mux.RUnlock()

	// no statement until first connected
	if c.productionLimitActivated == nil {
		return nil
	}

	// without a nominal reference the W limit cannot be expressed as a percent
	if c.productionNominalMax <= 0 {
		return nil
	}

	percent := 100
	if limitActive(c.productionLimitActivated) {
		// the EG may state a limit above the nominal production power
		percent = min(int(c.effectiveProductionLimit()/c.productionNominalMax*100), 100)
	}

	return &percent
}

// MaxConsumptionPower implements api.HEMS: nil until first connected,
// else failsafe limit in failsafe, else the active EG-supplied LPC limit, else 0.
func (c *EEBus) MaxConsumptionPower() *float64 {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.consumptionLimitActivated == nil {
		return nil
	}
	if !limitActive(c.consumptionLimitActivated) {
		return new(0.0)
	}
	return new(c.effectiveConsumptionLimit())
}

// MaxProductionPower implements api.HEMS: nil until first connected,
// else failsafe limit in failsafe, else the active EG-supplied LPP limit, else 0.
func (c *EEBus) MaxProductionPower() *float64 {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.productionLimitActivated == nil {
		return nil
	}
	if !limitActive(c.productionLimitActivated) {
		return new(0.0)
	}
	return new(c.effectiveProductionLimit())
}
