package fnn

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// NewFromConfig creates an FNN HEMS from generic config.
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Fnn, error) {
	cc := struct {
		MaxPower        float64 // TODO deprecated
		MaxDimPower     float64
		MaxCurtailPower float64
		W3              *plugin.Config
		S1              *plugin.Config
		S2              *plugin.Config
		W4              *plugin.Config
		Interval        time.Duration
	}{
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	w3G, err := cc.W3.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	w4G, err := cc.W4.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	// either dim or curtail must be configured
	if w3G == nil && w4G == nil {
		return nil, errors.New("must have either W3 or W4")
	}

	s1G, err := cc.S1.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	s2G, err := cc.S2.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	maxDimPower := math.Abs(cc.MaxDimPower)
	if w4G != nil && maxDimPower == 0 {
		return nil, errors.New("cannot have w4 without power limit")
	}

	maxCurtailPower := math.Abs(cc.MaxCurtailPower)
	if cc.MaxPower > 0 {
		maxCurtailPower = cc.MaxPower
	}

	return &Fnn{
		log:             util.NewLogger("fnn"),
		site:            site,
		maxDimPower:     maxDimPower,
		maxCurtailPower: maxCurtailPower,
		s1:              s1G,
		s2:              s2G,
		w3:              w3G,
		w4:              w4G,
		interval:        cc.Interval,
	}, nil
}

// Fnn implements the FNN HEMS logic for curtailment and dimming.
type Fnn struct {
	mu  sync.Mutex
	log *util.Logger

	site        site.API
	s1, s2, w3  func() (bool, error)
	w4          func() (bool, error)
	publishFunc func()

	maxDimPower     float64
	maxCurtailPower float64

	smartgridConsumptionID uint
	smartgridProductionID  uint

	consumptionLimit float64
	productionLimit  *float64

	interval time.Duration
}

func (c *Fnn) SetUpdated(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishFunc = f
}

// Run starts the FNN control loop.
func (c *Fnn) Run() {
	for range time.Tick(c.interval) {
		if err := c.runCurtail(); err != nil {
			c.log.ERROR.Println(err)
		}

		if err := c.runDim(); err != nil {
			c.log.ERROR.Println(err)
		}

		if c.publishFunc != nil {
			c.publishFunc()
		}
	}
}

// runCurtail evaluates curtailment rules and applies the appropriate limit.
// No-op if no curtail input is configured.
func (c *Fnn) runCurtail() error {
	if c.w3 == nil {
		return nil
	}

	rules := []struct {
		get  func() (bool, error)
		frac float64
	}{
		{get: c.w3, frac: 0.0},
		{get: c.s2, frac: 0.3},
		{get: c.s1, frac: 0.6},
	}

	for _, rule := range rules {
		if rule.get == nil {
			continue
		}

		active, err := rule.get()
		if err != nil {
			return err
		}

		if active {
			return c.setProductionLimit(rule.frac)
		}
	}

	// 100%
	return c.setProductionLimit(1.0)
}

// runDim evaluates the dimming rule and applies the dim limit.
// No-op if dim input is not configured.
func (c *Fnn) runDim() error {
	if c.w4 == nil {
		return nil
	}

	active, err := c.w4()
	if err != nil {
		return err
	}

	limit := 0.0
	if active {
		limit = c.maxDimPower
	}

	return c.setConsumptionLimit(limit)
}

// setProductionLimit applies the curtailment limit.
func (c *Fnn) setProductionLimit(frac float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := frac < 1.0

	c.productionLimit = nil
	if active {
		c.productionLimit = new(c.maxCurtailPower * frac)
	}

	limit := 0.0
	if c.productionLimit != nil {
		limit = *c.productionLimit
	}

	if err := smartgrid.UpdateSession(&c.smartgridProductionID, smartgrid.Curtail, c.site.GetGridPower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}

// setConsumptionLimit applies the dimming limit.
func (c *Fnn) setConsumptionLimit(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := limit > 0
	c.consumptionLimit = limit

	if err := smartgrid.UpdateSession(&c.smartgridConsumptionID, smartgrid.Dim, c.site.GetGridPower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}

var _ api.HEMS = (*Fnn)(nil)

// Dimmed implements api.HEMS.
func (c *Fnn) Dimmed() bool {
	if c.w4 == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consumptionLimit > 0
}

// Curtailed implements api.HEMS.
func (c *Fnn) Curtailed() bool {
	if c.w3 == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.productionLimit != nil
}

// MaxConsumptionPower implements api.HEMS.
func (c *Fnn) MaxConsumptionPower() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consumptionLimit
}

// MaxProductionPower implements api.HEMS.
func (c *Fnn) MaxProductionPower() *float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.productionLimit == nil {
		return nil
	}

	return new(*c.productionLimit)
}
