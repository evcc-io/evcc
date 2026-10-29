package custom

import (
	"context"
	"errors"
	"fmt"
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
	config.AddCtx(api.Custom, NewFromConfig)
}

// Custom implements a plugin-configurable HEMS.
type Custom struct {
	mu  sync.Mutex
	log *util.Logger

	site        site.API
	publishFunc func()

	maxConsumptionPower func() (float64, error)
	curtailedPercent    func() (int64, error)

	productionNominalMax float64
	interval             time.Duration

	smartgridConsumptionID uint
	smartgridProductionID  uint

	consumptionLimit  *float64
	productionPercent int // allowed feed-in percent (0..100), 100 = uncurtailed
}

// NewFromConfig creates a custom HEMS from generic config.
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Custom, error) {
	cc := struct {
		MaxConsumptionPower  *plugin.Config
		CurtailedPercent     *plugin.Config
		ProductionNominalMax float64
		Interval             time.Duration
	}{
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	maxConsumptionPowerG, err := cc.MaxConsumptionPower.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("max consumption power: %w", err)
	}

	curtailedPercentG, err := cc.CurtailedPercent.IntGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("curtailed percent: %w", err)
	}

	return NewCustom(site, maxConsumptionPowerG, curtailedPercentG, math.Abs(cc.ProductionNominalMax), cc.Interval)
}

// NewCustom creates a custom HEMS.
func NewCustom(site site.API, maxConsumptionPower func() (float64, error), curtailedPercent func() (int64, error), productionNominalMax float64, interval time.Duration) (*Custom, error) {
	if maxConsumptionPower == nil && curtailedPercent == nil {
		return nil, errors.New("must have either maxconsumptionpower or curtailedpercent")
	}

	if curtailedPercent != nil && productionNominalMax == 0 {
		return nil, errors.New("cannot have curtailedpercent without productionnominalmax")
	}

	c := &Custom{
		log:                  util.NewLogger("custom"),
		site:                 site,
		maxConsumptionPower:  maxConsumptionPower,
		curtailedPercent:     curtailedPercent,
		productionNominalMax: productionNominalMax,
		productionPercent:    100,
		interval:             interval,
	}

	// read the plugins once synchronously so limits are valid as soon as NewCustom returns
	if err := c.run(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Custom) SetUpdated(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishFunc = f
}

// Run starts the control loop. NewCustom already ran the first pass.
func (c *Custom) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}

		if c.publishFunc != nil {
			c.publishFunc()
		}
	}
}

func (c *Custom) run() error {
	return errors.Join(c.runDim(), c.runCurtail())
}

// runDim reads the consumption limit. No-op if not configured.
// The previous limit is retained if reading fails.
func (c *Custom) runDim() error {
	if c.maxConsumptionPower == nil {
		return nil
	}

	limit, err := c.maxConsumptionPower()
	if err != nil {
		return err
	}

	if limit < 0 {
		return fmt.Errorf("invalid consumption limit: %.0fW", limit)
	}

	c.setConsumptionLimit(limit)

	if err := smartgrid.UpdateSession(&c.smartgridConsumptionID, smartgrid.Dim, c.site.GetGridPower(), limit, limit > 0); err != nil {
		return fmt.Errorf("smartgrid session: %v", err)
	}

	return nil
}

// runCurtail reads the curtailment percentage. No-op if not configured.
// The previous percentage is retained if reading fails.
func (c *Custom) runCurtail() error {
	if c.curtailedPercent == nil {
		return nil
	}

	percent, err := c.curtailedPercent()
	if err != nil {
		return err
	}

	if percent < 0 || percent > 100 {
		return fmt.Errorf("invalid curtailment percent: %d", percent)
	}

	c.setProductionLimit(int(percent))

	active := percent < 100
	var limit float64
	if active {
		limit = float64(percent) / 100 * c.productionNominalMax
	}

	if err := smartgrid.UpdateSession(&c.smartgridProductionID, smartgrid.Curtail, c.site.GetGridPower(), limit, active); err != nil {
		return fmt.Errorf("smartgrid session: %v", err)
	}

	return nil
}

// setConsumptionLimit applies the dimming limit.
func (c *Custom) setConsumptionLimit(limit float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.consumptionLimit = nil
	if limit > 0 {
		c.consumptionLimit = &limit
	}
}

// setProductionLimit applies the curtailment limit.
func (c *Custom) setProductionLimit(percent int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.productionPercent = percent
}

var _ api.HEMS = (*Custom)(nil)

// CurtailedPercent implements api.HEMS, returning the allowed production percent.
func (c *Custom) CurtailedPercent() *int {
	if c.curtailedPercent == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return new(c.productionPercent)
}

// MaxConsumptionPower implements api.HEMS, returning the active wattage cap.
func (c *Custom) MaxConsumptionPower() *float64 {
	if c.maxConsumptionPower == nil {
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
func (c *Custom) MaxProductionPower() *float64 {
	if c.curtailedPercent == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.productionPercent >= 100 {
		return new(0.0)
	}

	return new(float64(c.productionPercent) / 100 * c.productionNominalMax)
}
