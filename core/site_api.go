package core

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
)

var _ site.API = (*Site)(nil)

// GetPrioritySoc returns the PrioritySoc
func (site *Site) GetPrioritySoc() float64 {
	site.Lock()
	defer site.Unlock()
	return site.PrioritySoc
}

// SetPrioritySoc sets the PrioritySoc
func (site *Site) SetPrioritySoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return errors.New("battery not configured")
	}

	site.PrioritySoc = soc
	site.publish("prioritySoc", site.PrioritySoc)

	return nil
}

// GetBufferSoc returns the BufferSoc
func (site *Site) GetBufferSoc() float64 {
	site.Lock()
	defer site.Unlock()
	return site.BufferSoc
}

// SetBufferSoc sets the BufferSoc
func (site *Site) SetBufferSoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return errors.New("battery not configured")
	}

	site.BufferSoc = soc
	site.publish("bufferSoc", site.BufferSoc)

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

// GetTariff returns the tariffs rates
func (site *Site) GetTariff(name string) (api.Rates, error) {
	site.Lock()
	defer site.Unlock()

	var tariff api.Tariff

	switch name {
	case "grid":
		tariff = site.tariffs.Grid
	case "feedin":
		tariff = site.tariffs.FeedIn
	case "planner":
		if tariff = site.tariffs.Planner; tariff == nil {
			tariff = site.tariffs.Grid
		}
	}

	if tariff == nil {
		return nil, errors.New("invalid tariff")
	}

	return tariff.Rates()
}
