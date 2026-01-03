package fnn

import (
	"context"
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/shared"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

type Fnn3 struct {
	log        *util.Logger
	root       api.Circuit
	s1, s2, w3 func() (bool, error)
	maxPower   float64
	interval   time.Duration
}

// NewFromConfig creates an Fnn3 HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Fnn3, error) {
	cc := struct {
		MaxPower float64
		S1       plugin.Config
		S2       plugin.Config
		W3       plugin.Config
		Interval time.Duration
	}{
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// get root circuit
	root := circuit.Root()
	if root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}

	// register LPP circuit if not already registered
	lpp, err := shared.GetOrCreateCircuit("lpp", "fnn-3")
	if err != nil {
		return nil, err
	}

	// wrap old root with new pc parent
	if err := root.Wrap(lpp); err != nil {
		return nil, err
	}
	site.SetCircuit(lpp)

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

	return NewFnn3(lpp, s1G, s2G, w3G, cc.MaxPower, cc.Interval)
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

	s2, err := c.s2()
	if err != nil {
		return err
	}

	if s2 {
		// 30%
		return c.curtail(0.3)
	}

	s1, err := c.s1()
	if err != nil {
		return err
	}

	if s1 {
		// 60%
		return c.curtail(0.6)
	}

	// 100%
	return c.curtail(1.0)
}

func (c *Fnn3) curtail(frac float64) error {
	c.root.Curtail(frac < 1.0)
	// TODO make ProductionNominalMax configurable (Site kWp)
	// c.root.SetMaxPower(c.maxPower*frac)
	return nil
}
