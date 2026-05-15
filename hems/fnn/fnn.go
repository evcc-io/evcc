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

type Fnn struct {
	mu  sync.Mutex
	log *util.Logger

	root       api.Circuit
	s1, s2, w3 func() (bool, error)
	w4         func() (bool, error)

	smartgridID    uint
	smartgridDimID uint
	limit          *float64
	maxPower       float64
	maxPowerDim    float64
	interval       time.Duration
}

 // NewFromConfig creates an FNN HEMS from generic config
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

// NewFnn creates Fnn HEMS
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

func boolGetter(ctx context.Context, cfg *plugin.Config) (func() (bool, error), error) {
	if cfg == nil {
		return nil, nil
	}

	return cfg.BoolGetter(ctx)
}

func (c *Fnn) Run() {
	for range time.Tick(c.interval) {
		if err := c.Update(); err != nil {
			c.log.ERROR.Println(err)
		}

		if err := c.runDim(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

func (c *Fnn) Update() error {
	if c.w3 != nil {
		w3, err := c.w3()
		if err != nil {
			return err
		}

		if w3 {
			// 0%
			return c.curtail(0.0)
		}
	}

	if c.s2 != nil {
		s2, err := c.s2()
		if err != nil {
			return err
		}

		if s2 {
			// 30%
			return c.curtail(0.3)
		}
	}

	if c.s1 != nil {
		s1, err := c.s1()
		if err != nil {
			return err
		}

		if s1 {
			// 60%
			return c.curtail(0.6)
		}
	}

	// 100%
	return c.curtail(1.0)
}

func (c *Fnn) runDim() error {
	if c.maxPowerDim <= 0 || c.w4 == nil {
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

func (c *Fnn) curtail(frac float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := frac < 1.0

	c.limit = nil
	if active {
		c.limit = new(c.maxPower * frac)
	}

	c.root.Curtail(active)
	// TODO make ProductionNominalMax configurable (Site kWp)
	// c.root.SetMaxPower(c.maxPower*frac)

	if err := smartgrid.UpdateSession(&c.smartgridID, smartgrid.Curtail, c.root.GetChargePower(), c.maxPower*frac, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}

func (c *Fnn) setDim(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := limit > 0

	c.root.Dim(active)
	c.root.SetMaxPower(limit)

	if err := smartgrid.UpdateSession(&c.smartgridDimID, smartgrid.Dim, c.root.GetChargePower(), limit, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}
