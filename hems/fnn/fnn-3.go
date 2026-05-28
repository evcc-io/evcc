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

type Fnn3 struct {
	mu  sync.Mutex
	log *util.Logger

	root       api.Circuit
	s1, s2, w3 func() (bool, error)

	smartgridID uint
	limit       *float64
	maxPower    float64
	interval    time.Duration

	// api.HEMS state mirrors what gets pushed to the root circuit; readers will
	// migrate to these fields and the circuit writes go away in a later step.
	curtailed *bool
}

// NewFromConfig creates an Fnn3 HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Fnn3, error) {
	cc := struct {
		MaxPower float64
		W3       plugin.Config
		S1       *plugin.Config
		S2       *plugin.Config
		Interval time.Duration
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

	// s1 getter
	s1G, err := cc.S1.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	s2G, err := cc.S2.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	w3G, err := cc.W3.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	return NewFnn3(gridcontrol, s1G, s2G, w3G, cc.MaxPower, cc.Interval)
}

// NewFnn3 creates Fnn3 HEMS
func NewFnn3(root api.Circuit, s1, s2, w3 func() (bool, error), maxPower float64, interval time.Duration) (*Fnn3, error) {
	c := &Fnn3{
		log:      util.NewLogger("fnn3"),
		root:     root,
		maxPower: maxPower,
		s1:       s1,
		s2:       s2,
		w3:       w3,
		interval: interval,
	}

	return c, nil
}

func (c *Fnn3) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

func (c *Fnn3) run() error {
	w3, err := c.w3()
	if err != nil {
		return err
	}

	if w3 {
		// 0%
		return c.curtail(0.0)
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

func (c *Fnn3) curtail(frac float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := frac < 1.0

	c.limit = nil
	if active {
		c.limit = new(c.maxPower * frac)
	}

	c.root.Curtail(active)
	c.curtailed = &active
	// TODO make ProductionNominalMax configurable (Site kWp)
	// c.root.SetMaxPower(c.maxPower*frac)

	if err := smartgrid.UpdateSession(&c.smartgridID, smartgrid.Curtail, c.root.GetChargePower(), c.maxPower*frac, active); err != nil {
		c.log.ERROR.Printf("smartgrid session: %v", err)
	}

	return nil
}

var _ api.HEMS = (*Fnn3)(nil)

// Dimmed implements api.HEMS
func (c *Fnn3) Dimmed() *bool {
	return nil
}

// Curtailed implements api.HEMS
func (c *Fnn3) Curtailed() *bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.curtailed
}

// MaxConsumptionPower implements api.HEMS
func (c *Fnn3) MaxConsumptionPower() float64 {
	return 0
}

// MaxProductionPower implements api.HEMS
func (c *Fnn3) MaxProductionPower() *float64 {
	return nil
}
