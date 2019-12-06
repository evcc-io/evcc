package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	currentPowerG provider.FloatGetter
}

// NewMeter creates a new charger
func NewMeter(currentPowerG provider.FloatGetter) api.Meter {
	return &Meter{
		currentPowerG: currentPowerG,
	}
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
