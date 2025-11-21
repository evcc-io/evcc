package relay

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

type Relay struct {
	log *util.Logger

	root     api.Circuit
	limit    func() (bool, error)
	maxPower float64
	interval time.Duration
}

// NewFromConfig creates an Relay HEMS from generic config
func NewFromConfig(ctx context.Context, other map[string]any, site site.API) (*Relay, error) {
	cc := struct {
		MaxPower float64
		Limit    plugin.Config
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

	// register LPC circuit if not already registered
	lpc, err := shared.GetOrCreateCircuit("lpc", "relay")
	if err != nil {
		return nil, err
	}

	// wrap old root with new pc parent
	if err := root.Wrap(lpc); err != nil {
		return nil, err
	}
	site.SetCircuit(lpc)

	// limit getter
	limitG, err := cc.Limit.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	return NewRelay(lpc, limitG, cc.MaxPower, cc.Interval)
}

// NewRelay creates Relay HEMS
func NewRelay(root api.Circuit, limit func() (bool, error), maxPower float64, interval time.Duration) (*Relay, error) {
	c := &Relay{
		log:      util.NewLogger("relay"),
		root:     root,
		maxPower: maxPower,
		limit:    limit,
		interval: interval,
	}

	return c, nil
}

func (c *Relay) Run() {
	for range time.Tick(c.interval) {
		if err := c.run(); err != nil {
			c.log.ERROR.Println(err)
		}
	}
}

func (c *Relay) run() error {
	limit, err := c.limit()
	if err != nil {
		return err
	}

	var power float64
	if limit {
		power = c.maxPower
	}

	c.root.Dim(limit)
	c.root.SetMaxPower(power)

	return nil
}
