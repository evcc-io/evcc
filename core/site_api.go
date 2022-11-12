package core

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
)

var _ site.API = (*Site)(nil)

// GetPrioritySoC returns the PrioritySoC
func (site *Site) GetPrioritySoC() float64 {
	site.Lock()
	defer site.Unlock()
	return site.PrioritySoC
}

// SetPrioritySoC sets the PrioritySoC
func (site *Site) SetPrioritySoC(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return errors.New("battery not configured")
	}

	site.PrioritySoC = soc
	site.publish("prioritySoC", site.PrioritySoC)

	return nil
}

// GetBufferSoC returns the BufferSoC
func (site *Site) GetBufferSoC() float64 {
	site.Lock()
	defer site.Unlock()
	return site.BufferSoC
}

// SetBufferSoC sets the BufferSoC
func (site *Site) SetBufferSoC(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return errors.New("battery not configured")
	}

	site.BufferSoC = soc
	site.publish("bufferSoC", site.BufferSoC)

	return nil
}

// GetResidualPower returns the ResidualPower
func (site *Site) GetResidualPower() float64 {
	site.Lock()
	defer site.Unlock()
	return site.ResidualPower
}

// SetResidualPower sets the ResidualPower
func (site *Site) SetResidualPower(power float64) error {
	site.Lock()
	defer site.Unlock()

	site.ResidualPower = power
	site.publish("residualPower", site.ResidualPower)

	return nil
}

// GetVehicles is the list of vehicles
func (site *Site) GetVehicles() []api.Vehicle {
	site.Lock()
	defer site.Unlock()
	return site.coordinator.GetVehicles()
}
