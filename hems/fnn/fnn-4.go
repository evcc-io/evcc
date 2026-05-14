package fnn

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

type Fnn4 struct {
	mu  sync.Mutex
	log *util.Logger

	root api.Circuit

	curtail *Fnn3

	w4          func() (bool, error)
	dimMaxPower float64
	interval    time.Duration

	smartgridDimID uint
}

// NewFnn4FromConfig creates an Fnn4 HEMS from generic config
func NewFnn4FromConfig(ctx context.Context, other map[string]any, site site.API) (*Fnn4, error) {
	cc := struct {
		MaxPower    float64
		DimMaxPower float64
		W4          plugin.Config
		W3          plugin.Config
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

	// fnn-3 inputs
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

	// fnn-4 additional dim input
	w4G, err := cc.W4.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	curtail, err := NewFnn3(gridcontrol, s1G, s2G, w3G, cc.MaxPower, cc.Interval)
	if err != nil {
		return nil, err
	}

	return NewFnn4(gridcontrol, curtail, w4G, cc.DimMaxPower, cc.Interval)
}

// NewFnn4 creates Fnn4 HEMS
func NewFnn4(root api.Circuit, curtail *Fnn3, w4 func() (bool, error), dimMaxPower float64, interval time.Duration) (*Fnn4, error) {
	c := &Fnn4{
		log:         util.NewLogger("fnn4"),
		root:        root,
		curtail:     curtail,
		w4:          w4,
		dimMaxPower: dimMaxPower,
		interval:    interval,
	}

	return c, nil
}

func (c *Fnn4) Run() {
	for range time.Tick(c.interval) {
		if err := c.curtail.Update(); err != nil {
			c.log.ERROR.Println(err)
		}

		if err := c.runDim(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

func (c *Fnn4) runDim() error {
	active, err := c.w4()
	if err != nil {
		return err
	}

	var limit float64
	if active {
		limit = c.dimMaxPower
	}

	return c.setDim(limit)
}

func (c *Fnn4) setDim(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := limit > 0

	c.root.Dim(active)
	c.root.SetMaxPower(limit)

	if err := smartgrid.UpdateSession(&c.smartgridDimID, smartgrid.Dim, c.root.GetChargePower(), limit, active); err != nil {
		return fmt.Errorf("smartgrid session: %v", err)
	}

	return nil
}
