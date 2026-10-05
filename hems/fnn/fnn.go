package fnn

import (
	"context"
	"errors"
	"math"
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
	config.AddCtx("fnn", NewFromConfig)
	config.AddCtx("fnn-3", NewFromConfig)
}

// NewFromConfig creates an FNN HEMS from generic config.
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Fnn, error) {
	cc := struct {
		MaxPower             float64 // TODO deprecated
		MaxDimPower          float64
		MaxCurtailPower      float64 // TODO deprecated
		ProductionNominalMax float64
		W3                   *plugin.Config
		S1                   *plugin.Config
		S2                   *plugin.Config
		W4                   *plugin.Config
		Interval             time.Duration
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

	productionNominalMax := math.Abs(cc.MaxCurtailPower)
	if cc.MaxPower > 0 {
		productionNominalMax = cc.MaxPower
	}
	// ProductionNominalMax supersedes deprecated MaxCurtailPower/MaxPower
	if cc.ProductionNominalMax > 0 {
		productionNominalMax = math.Abs(cc.ProductionNominalMax)
	}

	return NewFnn(site, math.Abs(cc.MaxDimPower), productionNominalMax, w3G, s1G, s2G, w4G, cc.Interval)
}

func NewFnn(site site.API, maxDimPower, productionNominalMax float64, w3G, s1G, s2G, w4G func() (bool, error), interval time.Duration) (*Fnn, error) {
	if w4G != nil && maxDimPower == 0 {
		return nil, errors.New("cannot have w4 without power limit")
	}

	c := &Fnn{
		log:                  util.NewLogger("fnn"),
		site:                 site,
		maxDimPower:          maxDimPower,
		productionNominalMax: productionNominalMax,
		s1:                   s1G,
		s2:                   s2G,
		w3:                   w3G,
		w4:                   w4G,
		productionPercent:    100,
		interval:             interval,
	}

	// read the relays once synchronously so limits are valid as soon as NewFnn returns
	if err := c.runCurtail(); err != nil {
		return nil, err
	}
	if err := c.runDim(); err != nil {
		return nil, err
	}

	return c, nil
}

// Fnn implements the FNN HEMS logic for curtailment and dimming.
type Fnn struct {
	mu  sync.Mutex
	log *util.Logger

	site        site.API
	s1, s2, w3  func() (bool, error)
	w4          func() (bool, error)
	publishFunc func()

	maxDimPower          float64
	productionNominalMax float64

	smartgridConsumptionID uint
	smartgridProductionID  uint

	consumptionLimit  *float64
	productionPercent int // allowed feed-in percent (0..100), 100 = uncurtailed

	interval time.Duration
}

func (c *Fnn) SetUpdated(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishFunc = f
}

// Run starts the FNN control loop. NewFnn already ran the first pass.
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
		get     func() (bool, error)
		percent int
	}{
		{get: c.w3, percent: 0},
		{get: c.s2, percent: 30},
		{get: c.s1, percent: 60},
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
			return c.setProductionLimit(rule.percent)
		}
	}

	// 100%
	return c.setProductionLimit(100)
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
func (c *Fnn) setProductionLimit(percent int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := percent < 100
	c.productionPercent = percent

	limit := 0.0
	if active {
		limit = float64(percent) / 100 * c.productionNominalMax
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
	c.consumptionLimit = nil
	if active {
		c.consumptionLimit = &limit
	}

	if err := smartgrid.UpdateSession(&c.smartgridConsumptionID, smartgrid.Dim, c.site.GetGridPower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}

var _ api.HEMS = (*Fnn)(nil)

// CurtailedPercent implements api.HEMS, returning the allowed production percent.
func (c *Fnn) CurtailedPercent() *int {
	if c.w3 == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return new(c.productionPercent)
}

// MaxConsumptionPower implements api.HEMS.
func (c *Fnn) MaxConsumptionPower() *float64 {
	if c.w4 == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.consumptionLimit == nil {
		return new(0.0)
	}
	return new(*c.consumptionLimit)
}

// MaxProductionPower implements api.HEMS.
func (c *Fnn) MaxProductionPower() *float64 {
	if c.w3 == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.productionPercent >= 100 {
		return new(0.0)
	}

	return new(float64(c.productionPercent) / 100 * c.productionNominalMax)
}
