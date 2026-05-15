package fnn

import (
	"context"
	"sync"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// NewFromConfig creates an FNN HEMS from generic config.
// The config struct fields align with EEBUS HEMS naming.
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Fnn, error) {
	cc := struct {
		MaxPower    float64
		MaxPowerDim float64
		W3          *plugin.Config
		W4          *plugin.Config
		S1          *plugin.Config
		S2          *plugin.Config
		Interval    time.Duration
	}{
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// setup grid control circuit
	gridcontrol, err := smartgrid.SetupCircuit()
	if err != nil {
		return nil, err
	}

	site.SetCircuit(gridcontrol)

	s1G, err := boolGetter(ctx, cc.S1)
	if err != nil {
		return nil, err
	}

	s2G, err := boolGetter(ctx, cc.S2)
	if err != nil {
		return nil, err
	}

	w3G, err := boolGetter(ctx, cc.W3)
	if err != nil {
		return nil, err
	}

	w4G, err := boolGetter(ctx, cc.W4)
	if err != nil {
		return nil, err
	}

	return NewFnn(gridcontrol, s1G, s2G, w3G, w4G, cc.MaxPower, cc.MaxPowerDim, cc.Interval)
}

// NewFnn creates a new Fnn HEMS instance.
func NewFnn(root api.Circuit, s1, s2, w3, w4 func() (bool, error), maxPower, maxPowerDim float64, interval time.Duration) (*Fnn, error) {
	c := &Fnn{
		log:         util.NewLogger("Fnn"),
		root:        root,
		maxPower:    maxPower,
		maxPowerDim: maxPowerDim,
		s1:          s1,
		s2:          s2,
		w3:          w3,
		w4:          w4,
		interval:    interval,
	}
	return c, nil
}

// boolGetter returns a boolean getter function for the given plugin config.
func boolGetter(ctx context.Context, cfg *plugin.Config) (func() (bool, error), error) {
	if cfg == nil {
		return func() (bool, error) { return false, nil }, nil
	}
	return cfg.BoolGetter(ctx)
}

// Fnn implements the FNN HEMS logic for curtailment and dimming.
type Fnn struct {
	mu  sync.Mutex
	log *util.Logger

	root       api.Circuit
	s1, s2, w3 func() (bool, error)
	w4         func() (bool, error)

	smartgridConsumptionId    uint
	consumptionLimit          ucapi.LoadLimit // LPC-041
	consumptionLimitActivated time.Time
	failsafeConsumptionLimit  float64

	smartgridProductionId    uint
	productionLimit          ucapi.LoadLimit
	productionLimitActivated time.Time
	failsafeProductionLimit  *float64

	// legacy fields
	limit       *float64
	maxPower    float64
	maxPowerDim float64
	interval    time.Duration
}

type curtailRule struct {
	getter   func() (bool, error)
	fraction float64
}

// Run starts the FNN control loop.
func (c *Fnn) Run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := c.Update(); err != nil {
			c.log.ERROR.Println(err)
		}

		if err := c.runDim(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

// Update evaluates curtailment rules and applies the appropriate limit.
func (c *Fnn) Update() error {
	rules := []curtailRule{
		{getter: c.w3, fraction: 0.0},
		{getter: c.s2, fraction: 0.3},
		{getter: c.s1, fraction: 0.6},
	}

	for _, rule := range rules {
		active, err := rule.getter()
		if err != nil {
			return err
		}

		if active {
			return c.curtail(rule.fraction)
		}
	}

	// 100%
	return c.curtail(1.0)
}

// runDim evaluates the dimming rule and applies the dim limit.
func (c *Fnn) runDim() error {
	if c.maxPowerDim <= 0 {
		return nil
	}

	active, err := c.w4()
	if err != nil {
		return err
	}

	var frac float64
	if active {
		frac = c.maxPowerDim
	}

	return c.setDim(frac)
}

// curtail applies the curtailment limit to the circuit.
func (c *Fnn) curtail(frac float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := frac < 1.0
	limit := c.maxPower * frac
	c.productionLimit = ucapi.LoadLimit{}
	c.productionLimitActivated = time.Now()

	c.limit = nil
	if active {
		c.limit = &limit
		c.failsafeProductionLimit = &limit
	} else {
		c.failsafeProductionLimit = nil
	}

	c.root.Curtail(active)
	// TODO make ProductionNominalMax configurable (Site kWp)
	// c.root.SetMaxPower(c.maxPower*frac)

	if err := smartgrid.UpdateSession(&c.smartgridProductionId, smartgrid.Curtail, c.root.GetChargePower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}

// setDim applies the dimming limit to the circuit.
func (c *Fnn) setDim(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := limit > 0
	c.consumptionLimit = ucapi.LoadLimit{}
	c.consumptionLimitActivated = time.Now()
	c.failsafeConsumptionLimit = limit

	c.root.Dim(active)
	c.root.SetMaxPower(limit)

	if err := smartgrid.UpdateSession(&c.smartgridConsumptionId, smartgrid.Dim, c.root.GetChargePower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}
