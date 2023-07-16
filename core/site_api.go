package core

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
)

var _ site.API = (*Site)(nil)

const (
	GridTariff    = "grid"
	FeedinTariff  = "feedin"
	PlannerTariff = "planner"
)

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

// GetTariff returns the respective tariff if configured or nil
func (site *Site) GetTariff(tariff string) api.Tariff {
	site.Lock()
	defer site.Unlock()

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

		case site.tariffs.Grid != nil && site.tariffs.Grid.Type() == api.TariffTypePriceDynamic:
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

func (site *Site) GetPowerData(year int, month int, day int, offset int) []api.PowerDataItem {
	result := site.db.GetPowerStatesForDay(uint16(year), uint16(month), uint16(day), uint16(offset))
	data := []api.PowerDataItem{}
	for i := 0; i < len(result); i++ {
		var item api.PowerDataItem
		item.TimePoint = fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:00Z", result[i].Year, result[i].Month, result[i].Day, result[i].Hour, result[i].Minute)
		item.FromPvs = fmt.Sprintf("%d", uint16(result[i].FromPvs))
		item.FromStorage = fmt.Sprintf("%d", uint16(result[i].FromStorage))
		item.FromGrid = fmt.Sprintf("%d", uint16(result[i].FromGrid))
		item.ToGrid = fmt.Sprintf("%d", uint16(result[i].ToGrid))
		item.ToHouse = fmt.Sprintf("%d", uint16(result[i].ToHouse))
		item.ToStorage = fmt.Sprintf("%d", uint16(result[i].ToStorage))
		item.ToHeating = fmt.Sprintf("%d", uint16(result[i].ToHeating))
		item.ToCars = fmt.Sprintf("%d", uint16(result[i].ToCars))
		item.BatterySoC = fmt.Sprintf("%d", uint16(result[i].BatterySoC))
		data = append(data, item)
	}
	return data

}
