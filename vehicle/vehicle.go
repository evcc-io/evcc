package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

type embed struct {
	title    string
	capacity int64
}

// Title implements the Vehicle.Title interface
func (m *embed) Title() string {
	return m.title
}

// Capacity implements the Vehicle.Capacity interface
func (m *embed) Capacity() int64 {
	return m.capacity
}

// Vehicle is an api.Vehicle implementation with configurable getters and setters.
type Vehicle struct {
	*embed
	chargeG func() (float64, error)
}

func init() {
	registry.Add("default", NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new Vehicle
func NewConfigurableFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title    string
		Capacity int64
		Charge   provider.Config `validate:"required"`
		Cache    time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	getter, err := provider.NewFloatGetterFromConfig(cc.Charge)
	if err != nil {
		return nil, err
	}

	if cc.Cache > 0 {
		getter = provider.NewCached(getter, cc.Cache).FloatGetter()
	}

	v := &Vehicle{
		embed:   &embed{cc.Title, cc.Capacity},
		chargeG: getter,
	}

	return v, nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Vehicle) ChargeState() (float64, error) {
	return m.chargeG()
}
