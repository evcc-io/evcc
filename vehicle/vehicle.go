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
	chargeG provider.FloatGetter
}

// NewConfigurableFromConfig creates a new Vehicle
func NewConfigurableFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title    string
		Capacity int64
		Charge   provider.Config
		Cache    time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	getter := provider.NewFloatGetterFromConfig(log, cc.Charge)
	if cc.Cache > 0 {
		getter = provider.NewCached(log, getter, cc.Cache).FloatGetter()
	}

	return &Vehicle{
		embed:   &embed{cc.Title, cc.Capacity},
		chargeG: getter,
	}
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Vehicle) ChargeState() (float64, error) {
	return m.chargeG()
}
