package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// MeterEnergy is an api.MeterEnergy implementation with configurable getters and setters.
type MeterEnergy struct {
	totalEnergyG provider.FloatGetter
}

// NewMeterEnergy creates a new charger
func NewMeterEnergy(totalEnergyG provider.FloatGetter) api.MeterEnergy {
	return &MeterEnergy{
		totalEnergyG: totalEnergyG,
	}
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *MeterEnergy) TotalEnergy() (float64, error) {
	return m.totalEnergyG()
}
