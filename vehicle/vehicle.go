package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

const (
	tokenValidMargin = 10 * time.Second // safety margin for api token validity
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
		Cache    time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	return &Vehicle{
		embed:   &embed{cc.Title, cc.Capacity},
		chargeG: provider.NewCached(provider.NewFloatGetterFromConfig(cc.Charge), cc.Cache).FloatGetter(),
	}
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Vehicle) ChargeState() (float64, error) {
	return m.chargeG()
}
