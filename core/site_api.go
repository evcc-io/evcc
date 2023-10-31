package core

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
)

var _ site.API = (*Site)(nil)

var ErrBatteryNotConfigured = errors.New("battery not configured")

const (
	GridTariff    = "grid"
	FeedinTariff  = "feedin"
	PlannerTariff = "planner"
)

// GetPrioritySoc returns the PrioritySoc
func (site *Site) GetPrioritySoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.prioritySoc
}

// SetPrioritySoc sets the PrioritySoc
func (site *Site) SetPrioritySoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return ErrBatteryNotConfigured
	}

	site.log.DEBUG.Println("set priority soc:", soc)

	if site.prioritySoc != soc {
		site.prioritySoc = soc
		settings.SetFloat(keys.PrioritySoc, site.prioritySoc)
		site.publish(keys.PrioritySoc, site.prioritySoc)
	}

	return nil
}

// GetBufferSoc returns the BufferSoc
func (site *Site) GetBufferSoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.bufferSoc
}

// SetBufferSoc sets the BufferSoc
func (site *Site) SetBufferSoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return ErrBatteryNotConfigured
	}

	site.log.DEBUG.Println("set buffer soc:", soc)

	if site.bufferSoc != soc {
		site.bufferSoc = soc
		settings.SetFloat(keys.BufferSoc, site.bufferSoc)
		site.publish(keys.BufferSoc, site.bufferSoc)
	}

	return nil
}

// GetBufferStartSoc returns the BufferStartSoc
func (site *Site) GetBufferStartSoc() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.bufferStartSoc
}

// SetBufferStartSoc sets the BufferStartSoc
func (site *Site) SetBufferStartSoc(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if len(site.batteryMeters) == 0 {
		return ErrBatteryNotConfigured
	}

	site.log.DEBUG.Println("set buffer start soc:", soc)

	if site.bufferStartSoc != soc {
		site.bufferStartSoc = soc
		settings.SetFloat(keys.BufferStartSoc, site.bufferStartSoc)
		site.publish(keys.BufferStartSoc, site.bufferStartSoc)
	}

	return nil
}

// GetResidualPower returns the ResidualPower
func (site *Site) GetResidualPower() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.ResidualPower
}

// SetResidualPower sets the ResidualPower
func (site *Site) SetResidualPower(power float64) error {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set residual power:", power)

	if site.ResidualPower != power {
		site.ResidualPower = power
		site.publish(keys.ResidualPower, site.ResidualPower)
	}

	return nil
}

// GetSmartCostLimit returns the SmartCostLimit
func (site *Site) GetSmartCostLimit() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.SmartCostLimit
}

// SetSmartCostLimit sets the SmartCostLimit
func (site *Site) SetSmartCostLimit(val float64) error {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set smart cost limit:", val)

	if site.SmartCostLimit != val {
		site.SmartCostLimit = val
		settings.SetFloat(keys.SmartCostLimit, site.SmartCostLimit)
		site.publish(keys.SmartCostLimit, site.SmartCostLimit)
	}

	return nil
}

// GetVehicles returns the vehicles proxy
func (site *Site) Vehicles() site.Vehicles {
	vv := &vehicles{log: site.log}
	return vv
}

// GetTariff returns the respective tariff if configured or nil
func (site *Site) GetTariff(tariff string) api.Tariff {
	site.RLock()
	defer site.RUnlock()

	switch tariff {
	case GridTariff:
		return site.tariffs.Grid

	case FeedinTariff:
		return site.tariffs.FeedIn

	case PlannerTariff:
		switch {
		case site.tariffs.Planner != nil:
			// prio 0: manually set planner tariff
			return site.tariffs.Planner

		case site.tariffs.Grid != nil && site.tariffs.Grid.Type() == api.TariffTypePriceForecast:
			// prio 1: dynamic grid tariff
			return site.tariffs.Grid

		case site.tariffs.Co2 != nil:
			// prio 2: co2 tariff
			return site.tariffs.Co2

		default:
			// prio 3: static grid tariff
			return site.tariffs.Grid
		}

	default:
		return nil
	}
}
