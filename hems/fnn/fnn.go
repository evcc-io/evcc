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
		MaxCurtailPower *float64
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

	// setup grid control circuit
	gridcontrol, err := smartgrid.SetupCircuit()
	if err != nil {
		return nil, err
	}

	site.SetCircuit(gridcontrol)

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

	var maxCurtailPower *float64
	switch {
	case cc.MaxCurtailPower != nil:
		maxCurtailPower = new(math.Abs(*cc.MaxCurtailPower))
	case cc.MaxPower > 0:
		// fnn-3 backwards compatibility: legacy MaxPower was the PV/curtail cap
		maxCurtailPower = new(math.Abs(cc.MaxPower))
	}

	return &Fnn{
		log:             util.NewLogger("fnn"),
		root:            gridcontrol,
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

	root       api.Circuit
	s1, s2, w3 func() (bool, error)
	w4         func() (bool, error)

	smartgridConsumptionID uint
	smartgridProductionID  uint

	maxDimPower     float64
	maxCurtailPower *float64
	interval        time.Duration
}

// Run starts the FNN control loop.
func (c *Fnn) Run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := c.runCurtail(); err != nil {
			c.log.ERROR.Println(err)
		}

		if err := c.runDim(); err != nil {
			c.log.ERROR.Println(err)
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
		if c.maxDimPower <= 0 {
			return errors.New("dim active but no limit configured")
		}

		limit = c.maxDimPower
	}

	return c.setConsumptionLimit(limit)
}

func (c *Fnn) applyMode(id *uint, typ smartgrid.Type, active bool, limit float64, applyRoot func()) {
	applyRoot()

	if err := smartgrid.UpdateSession(id, typ, c.root.GetChargePower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}
}

// setProductionLimit applies the curtailment limit to the circuit.
func (c *Fnn) setProductionLimit(frac float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit := 0.0
	active := frac < 1.0

	if active {
		if c.maxCurtailPower == nil {
			return errors.New("curtail active but no limit configured")
		}

		limit = *c.maxCurtailPower * frac
	}

	c.applyMode(&c.smartgridProductionID, smartgrid.Curtail, active, limit, func() {
		c.root.Curtail(active)
		// TODO make ProductionNominalMax configurable (Site kWp)
		// c.root.SetMaxPower(c.maxPower*frac)
	})

	return nil
}

// setConsumptionLimit applies the dimming limit to the circuit.
func (c *Fnn) setConsumptionLimit(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := limit > 0

	c.applyMode(&c.smartgridConsumptionID, smartgrid.Dim, active, limit, func() {
		c.root.Dim(active)
		c.root.SetMaxPower(limit)
	})

	return nil
}
