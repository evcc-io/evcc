package core

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/tariff"
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
	settings.SetFloat("site.prioritySoc", site.PrioritySoc)
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
	settings.SetFloat("site.bufferSoc", site.BufferSoc)
	site.publish("bufferSoc", site.BufferSoc)

	return nil
}

// GetBufferStartSoc returns the BufferStartSoc
func (site *Site) GetBufferStartSoc() float64 {
	site.Lock()
	defer site.Unlock()
	return site.BufferStartSoc
}

// SetBufferStartSoc sets the BufferStartSoc
func (site *Site) SetBufferStartSoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return errors.New("battery not configured")
	}

	site.BufferStartSoc = soc
	settings.SetFloat("site.bufferStartSoc", site.BufferStartSoc)
	site.publish("bufferStartSoc", site.BufferStartSoc)

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

// GetSmartCostLimit returns the SmartCostLimit
func (site *Site) GetSmartCostLimit() float64 {
	site.Lock()
	defer site.Unlock()
	return site.SmartCostLimit
}

// SetSmartCostLimit sets the SmartCostLimit
func (site *Site) SetSmartCostLimit(val float64) error {
	site.Lock()
	defer site.Unlock()

	site.SmartCostLimit = val
	settings.SetFloat("site.smartCostLimit", site.SmartCostLimit)
	site.publish("smartCostLimit", site.SmartCostLimit)

	return nil
}

// GetVehicles is the list of vehicles
func (site *Site) GetVehicles() []api.Vehicle {
	site.Lock()
	defer site.Unlock()
	return site.coordinator.GetVehicles()
}

func (site *Site) maxChargePower() float64 {
	var res float64
	for _, lp := range site.loadpoints {
		res += lp.GetMaxPower()
	}
	return res
}

// GetTariff returns the respective tariff if configured or nil
func (site *Site) GetTariff(name string, adjusted bool) api.Tariff {
	site.Lock()
	defer site.Unlock()

	t := site.tariffs.Get(name)
	if t == nil {
		return nil
	}

	if adjusted {
		gen := site.tariffs.Get(tariff.Generation)
		if gen == nil {
			return nil
		}

		// merge generation power
		if t.Type() == api.TariffTypeCo2 {
			return tariff.NewAdjusted(t, gen, site.maxChargePower())
		}
	}

	return t
}
