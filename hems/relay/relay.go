package relay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

type Relay struct {
	mu  sync.Mutex
	log *util.Logger

	root        api.Circuit
	w1          func() (bool, error)
	passthrough func(bool) error

	smartgridID uint
	limit       *float64
	maxPower    float64
	interval    time.Duration
}

// NewFromConfig creates an Relay HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Relay, error) {
	cc := struct {
		MaxPower    float64
		Limit       plugin.Config
		Passthrough *plugin.Config
		Interval    time.Duration
	}{
		Interval: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// setup grid control circuit
	gridcontrol, err := smartgrid.SetupCircuit("relay")
	if err != nil {
		return nil, err
	}

	site.SetCircuit(gridcontrol)

	// limit getter
	limitG, err := cc.Limit.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	passthroughS, err := cc.Passthrough.BoolSetter(ctx, "dim")
	if err != nil {
		return nil, err
	}

	return NewRelay(gridcontrol, limitG, passthroughS, cc.MaxPower, cc.Interval)
}

// NewRelay creates Relay HEMS
func NewRelay(root api.Circuit, w1 func() (bool, error), passthrough func(bool) error, maxPower float64, interval time.Duration) (*Relay, error) {
	c := &Relay{
		log:         util.NewLogger("relay"),
		root:        root,
		passthrough: passthrough,
		maxPower:    maxPower,
		w1:          w1,
		interval:    interval,
	}

	return c, nil
}

var _ hems.API = (*Relay)(nil)

func (c *Relay) ConsumptionLimit() *float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.limit
}

// ProductionLimit implements hems.API
func (c *Relay) ProductionLimit() *float64 {
	return nil
}

func (c *Relay) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

func (c *Relay) run() error {
	active, err := c.w1()
	if err != nil {
		return err
	}

	var limit float64
	if active {
		limit = c.maxPower
	}

	if err := c.setLimited(limit); err != nil {
		return err
	}

	if err := smartgrid.UpdateSession(&c.smartgridID, smartgrid.Dim, c.root.GetChargePower(), limit, active); err != nil {
		return fmt.Errorf("smartgrid session: %v", err)
	}

	return nil
}

func (c *Relay) setLimited(limit float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.limit = nil
	if limit > 0 {
		c.limit = new(limit)
	}

	c.root.Dim(limit > 0)
	c.root.SetMaxPower(limit)

	if c.passthrough != nil {
		if err := c.passthrough(limit > 0); err != nil {
			return fmt.Errorf("passthrough failed: %w", err)
		}
	}

	return nil
}
