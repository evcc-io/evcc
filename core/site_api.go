package core

import (
	"errors"
	"strings"

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

// GetTitle returns the title
func (site *Site) GetTitle() string {
	site.Lock()
	defer site.Unlock()
	return site.Title
}

// SetTitle sets the title
func (site *Site) SetTitle(title string) {
	site.Lock()
	defer site.Unlock()

	site.Title = title
	site.publish("siteTitle", title)
	settings.SetString("site.title", title)
}

// GetGridMeterRef returns the GridMeterRef
func (site *Site) GetGridMeterRef() string {
	site.Lock()
	defer site.Unlock()
	return site.Meters.GridMeterRef
}

// SetGridMeterRef sets the GridMeterRef
func (site *Site) SetGridMeterRef(ref string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.GridMeterRef = ref
	// site.publish("siteGridMeterRef", meter)
	settings.SetString("site.grid", ref)
	settings.Persist()
}

// GetPVMeterRef returns the PvMeterRef
func (site *Site) GetPVMeterRef() []string {
	site.Lock()
	defer site.Unlock()
	return site.Meters.PVMetersRef
}

// SetPVMeterRef sets the PvMeterRef
func (site *Site) SetPVMeterRef(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.PVMetersRef = ref
	// site.publish("siteGridMeterRef", meter)
	settings.SetString("site.pv", strings.Join(ref, ","))
	settings.Persist()
}

// GetBatteryMeterRef returns the BatteryMeterRef
func (site *Site) GetBatteryMeterRef() []string {
	site.Lock()
	defer site.Unlock()
	return site.Meters.BatteryMetersRef
}

// SetBatteryMeterRef sets the BatteryMeterRef
func (site *Site) SetBatteryMeterRef(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.BatteryMetersRef = ref
	// site.publish("siteGridMeterRef", meter)
	settings.SetString("site.battery", strings.Join(ref, ","))
	settings.Persist()
}

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
