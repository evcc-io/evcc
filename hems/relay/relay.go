package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/config"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

func init() {
	config.AddCtx("relay", NewFromConfig)
}

type Relay struct {
	mu  sync.Mutex
	log *util.Logger

	site        site.API
	w1          func() (bool, error)
	passthrough func(bool) error
	publishFunc func()

	smartgridID uint
	limit       *float64
	maxPower    float64
	interval    time.Duration

	failsafeConsumptionLimit float64
	failsafeDurationMinimum  time.Duration
	failsafeActive           bool
	failsafeSince            time.Time
}

// NewFromConfig creates an Relay HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Relay, error) {
	cc := struct {
		MaxPower                            float64
		Limit                               plugin.Config
		Passthrough                         *plugin.Config
		Interval                            time.Duration
		FailsafeConsumptionActivePowerLimit float64
		FailsafeDurationMinimum             time.Duration
	}{
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.FailsafeConsumptionActivePowerLimit < 0 {
		return nil, errors.New("failsafe consumption limit cannot be negative")
	}

	// limit getter
	limitG, err := cc.Limit.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	passthroughS, err := cc.Passthrough.BoolSetter(ctx, "dim")
	if err != nil {
		return nil, err
	}

	return NewRelay(
		site,
		limitG,
		passthroughS,
		cc.MaxPower,
		cc.Interval,
		cc.FailsafeConsumptionActivePowerLimit,
		cc.FailsafeDurationMinimum,
	)
}

// NewRelay creates Relay HEMS
func NewRelay(site site.API, w1 func() (bool, error), passthrough func(bool) error, maxPower float64, interval time.Duration, failsafeConsumptionLimit float64, failsafeDurationMinimum time.Duration) (*Relay, error) {
	c := &Relay{
		log:                      util.NewLogger("relay"),
		site:                     site,
		passthrough:              passthrough,
		maxPower:                 maxPower,
		w1:                       w1,
		interval:                 interval,
		failsafeConsumptionLimit: failsafeConsumptionLimit,
		failsafeDurationMinimum:  failsafeDurationMinimum,
	}

	if maxPower == 0 {
		return nil, errors.New("missing power limit")
	}
	if failsafeDurationMinimum < 0 {
		return nil, errors.New("failsafe duration cannot be negative")
	}

	// read the relay once synchronously so the limit is valid as soon as NewRelay returns
	if err := c.run(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Relay) SetUpdated(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishFunc = f
}

// Run starts the relay control loop. NewRelay already ran the first pass.
func (c *Relay) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}

		if c.publishFunc != nil {
			c.publishFunc()
		}
	}
}

func (c *Relay) run() error {
	active, err := c.w1()
	if err != nil {
		if !c.hasFailsafe() {
			return err
		}
		c.mu.Lock()
		if !c.failsafeActive {
			c.log.WARN.Printf("limit read error, entering failsafe mode: %v", err)
			c.failsafeActive = true
			c.failsafeSince = time.Now()
		}
		c.mu.Unlock()
		return c.setConsumptionLimit(c.failsafeConsumptionLimit)
	}

	c.mu.Lock()
	inFailsafe := false
	if c.failsafeActive {
		if time.Since(c.failsafeSince) >= c.failsafeDurationMinimum {
			c.log.DEBUG.Println("leaving failsafe mode")
			c.failsafeActive = false
		} else {
			inFailsafe = true
		}
	}
	c.mu.Unlock()

	if inFailsafe {
		return nil
	}

	var limit float64
	if active {
		limit = c.maxPower
	}

	if err := c.setConsumptionLimit(limit); err != nil {
		return err
	}

	if err := smartgrid.UpdateSession(&c.smartgridID, smartgrid.Dim, c.site.GetGridPower(), limit, active); err != nil {
		return fmt.Errorf("smartgrid session: %v", err)
	}

	return nil
}

func (c *Relay) hasFailsafe() bool {
	return c.failsafeConsumptionLimit > 0
}

func (c *Relay) setConsumptionLimit(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.limit = nil
	if limit > 0 {
		c.limit = new(limit)
	}

	if c.passthrough != nil {
		if err := c.passthrough(limit > 0); err != nil {
			return fmt.Errorf("passthrough failed: %w", err)
		}
	}

	return nil
}

var _ api.HEMS = (*Relay)(nil)

// CurtailedPercent implements api.HEMS. Relay does not curtail production and
// hence makes no statement.
func (c *Relay) CurtailedPercent() *int {
	return nil
}

// MaxConsumptionPower implements api.HEMS, returning the active wattage cap.
func (c *Relay) MaxConsumptionPower() *float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.limit == nil {
		return new(0.0)
	}
	return new(*c.limit)
}

// MaxProductionPower implements api.HEMS. Scaffolding only.
func (c *Relay) MaxProductionPower() *float64 {
	return nil
}
