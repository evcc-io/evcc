package relay

import (
	"context"
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type Relay struct {
	log *util.Logger

	root     api.Circuit
	limit    func() (bool, error)
	maxPower float64
}

// New creates an Relay HEMS from generic config
func New(ctx context.Context, other map[string]interface{}, site site.API) (*Relay, error) {
	var cc struct {
		MaxPower float64
		Limit    plugin.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// get root circuit
	root := circuit.Root()
	if root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}

	// create new root circuit for LPC
	lpc, err := circuit.New(util.NewLogger("lpc"), "relay", 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	// register LPC-Circuit for use in config, if not already registered
	if _, err := config.Circuits().ByName("lpc"); err != nil {
		_ = config.Circuits().Add(config.NewStaticDevice(config.Named{Name: "lpc"}, api.Circuit(lpc)))
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

	return NewRelay(lpc, limitG, cc.MaxPower)
}

// NewRelay creates Relay HEMS
func NewRelay(root api.Circuit, limit func() (bool, error), maxPower float64) (*Relay, error) {
	c := &Relay{
		log:      util.NewLogger("relay"),
		root:     root,
		maxPower: maxPower,
		limit:    limit,
	}

	return c, nil
}

func (c *Relay) Run() {
	for range time.Tick(10 * time.Second) {
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

	c.root.SetMaxPower(power)

	return nil
}
