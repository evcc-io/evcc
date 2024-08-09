package relay

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/provider"
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
func New(other map[string]interface{}, site site.API) (*Relay, error) {
	var cc struct {
		MaxPower float64
		Circuit  string
		Limit    provider.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	circuit, err := config.Circuits().ByName(cc.Circuit)
	if err != nil {
		return nil, fmt.Errorf("circuit: %w", err)
	}

	// limit getter
	limitG, err := provider.NewBoolGetterFromConfig(cc.Limit)
	if err != nil {
		return nil, err
	}

	return NewRelay(circuit.Instance(), limitG, cc.MaxPower)
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
