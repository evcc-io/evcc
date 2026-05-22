package fnn

import (
	"context"
	"sync"
	"time"

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
		MaxPower        float64
		MaxCurtailPower float64
		MaxPowerDim     float64
		MaxDimPower     float64
		W3              *plugin.Config
		W4              *plugin.Config
		S1              *plugin.Config
		S2              *plugin.Config
		Interval        time.Duration
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

	s1G := func() (bool, error) { return false, nil }
	if cc.S1 != nil {
		s1G, err = cc.S1.BoolGetter(ctx)
		if err != nil {
			return nil, err
		}
	}

	s2G := func() (bool, error) { return false, nil }
	if cc.S2 != nil {
		s2G, err = cc.S2.BoolGetter(ctx)
		if err != nil {
			return nil, err
		}
	}

	w3G := func() (bool, error) { return false, nil }
	if cc.W3 != nil {
		w3G, err = cc.W3.BoolGetter(ctx)
		if err != nil {
			return nil, err
		}
	}

	w4G := func() (bool, error) { return false, nil }
	if cc.W4 != nil {
		w4G, err = cc.W4.BoolGetter(ctx)
		if err != nil {
			return nil, err
		}
	}

	maxCurtailPower := cc.MaxCurtailPower
	if maxCurtailPower <= 0 {
		maxCurtailPower = cc.MaxPower
	}

	maxDimPower := cc.MaxDimPower
	if maxDimPower <= 0 {
		if cc.MaxPowerDim > 0 {
			maxDimPower = cc.MaxPowerDim
		} else if cc.MaxPower > 0 {
			maxDimPower = cc.MaxPower
		}
	}

	return &Fnn{
		log:         util.NewLogger("fnn"),
		root:        gridcontrol,
		maxPower:    maxCurtailPower,
		maxPowerDim: maxDimPower,
		s1:          s1G,
		s2:          s2G,
		w3:          w3G,
		w4:          w4G,
		interval:    cc.Interval,
	}, nil
}

// Fnn implements the FNN HEMS logic for curtailment and dimming.
type Fnn struct {
	mu  sync.Mutex
	log *util.Logger

	root       api.Circuit
	s1, s2, w3 func() (bool, error)
	w4         func() (bool, error)

	smartgridConsumptionID uint
	smartgridProductionID  uint

	maxPower    float64
	maxPowerDim float64
	interval    time.Duration

	lastCurtailActive bool
	lastCurtailLimit  float64
	lastCurtailSource string
	lastDimActive     bool
	lastDimLimit      float64
	lastDimSource     string
	curtailInit       bool
	dimInit           bool
}

type curtailRule struct {
	getter   func() (bool, error)
	fraction float64
	source   string
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
		{getter: c.w3, fraction: 0.0, source: "w3"},
		{getter: c.s2, fraction: 0.3, source: "s2"},
		{getter: c.s1, fraction: 0.6, source: "s1"},
	}

	for _, rule := range rules {
		active, err := rule.getter()
		if err != nil {
			return err
		}

		if active {
			return c.curtail(rule.fraction, rule.source)
		}
	}

	// 100%
	return c.curtail(1.0, "none")
}

// runDim evaluates the dimming rule and applies the dim limit.
func (c *Fnn) runDim() error {
	active, err := c.w4()
	if err != nil {
		return err
	}

	if c.maxPowerDim <= 0 {
		if active {
			c.log.WARN.Printf("dim active but no limit configured (maxDimPower/maxPowerDim)")
		}

		return nil
	}

	limit := 0.0
	if active {
		limit = c.maxPowerDim
	}

	return c.setDim(limit, "w4")
}

func (c *Fnn) applyMode(id *uint, typ smartgrid.Type, active bool, limit float64, applyRoot func()) {
	applyRoot()

	if err := smartgrid.UpdateSession(id, typ, c.root.GetChargePower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}
}

// curtail applies the curtailment limit to the circuit.
func (c *Fnn) curtail(frac float64, source string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := frac < 1.0
	if active && c.maxPower <= 0 {
		c.log.WARN.Printf("curtail active but no limit configured (maxCurtailPower/maxPower)")
		active = false
	}

	limit := c.maxPower * frac
	if !c.curtailInit {
		c.curtailInit = true
		if !active && source == "none" {
			c.lastCurtailActive = active
			c.lastCurtailLimit = limit
			c.lastCurtailSource = source
			c.applyMode(&c.smartgridProductionID, smartgrid.Curtail, active, limit, func() {
				c.root.Curtail(active)
				// TODO make ProductionNominalMax configurable (Site kWp)
				// c.root.SetMaxPower(c.maxPower*frac)
			})
			return nil
		}
	}

	if c.lastCurtailActive != active || c.lastCurtailLimit != limit || c.lastCurtailSource != source {
		c.log.DEBUG.Printf("curtail: source=%s active=%t fraction=%.2f limit=%.0fW", source, active, frac, limit)
		c.lastCurtailActive = active
		c.lastCurtailLimit = limit
		c.lastCurtailSource = source
	}

	c.applyMode(&c.smartgridProductionID, smartgrid.Curtail, active, limit, func() {
		c.root.Curtail(active)
		// TODO make ProductionNominalMax configurable (Site kWp)
		// c.root.SetMaxPower(c.maxPower*frac)
	})

	return nil
}

// setDim applies the dimming limit to the circuit.
func (c *Fnn) setDim(limit float64, source string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := limit > 0
	if !c.dimInit {
		c.dimInit = true
		if !active {
			c.lastDimActive = active
			c.lastDimLimit = limit
			c.lastDimSource = source
			c.applyMode(&c.smartgridConsumptionID, smartgrid.Dim, active, limit, func() {
				c.root.Dim(active)
				c.root.SetMaxPower(limit)
			})
			return nil
		}
	}

	if c.lastDimActive != active || c.lastDimLimit != limit || c.lastDimSource != source {
		c.log.DEBUG.Printf("dim: source=%s active=%t limit=%.0fW", source, active, limit)
		c.lastDimActive = active
		c.lastDimLimit = limit
		c.lastDimSource = source
	}

	c.applyMode(&c.smartgridConsumptionID, smartgrid.Dim, active, limit, func() {
		c.root.Dim(active)
		c.root.SetMaxPower(limit)
	})

	return nil
}
