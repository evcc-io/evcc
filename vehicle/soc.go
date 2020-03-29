package vehicle

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
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
func NewConfigurableFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title    string
		Capacity int64
		Charge   *provider.Config
	}{}
	api.DecodeOther(log, other, &cc)

	return &Vehicle{
		embed:   &embed{cc.Title, cc.Capacity},
		chargeG: provider.NewFloatGetterFromConfig(cc.Charge),
	}
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Vehicle) ChargeState() (float64, error) {
	return m.chargeG()
}
