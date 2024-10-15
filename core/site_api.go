package core

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
)

var _ site.API = (*Site)(nil)

var ErrBatteryNotConfigured = errors.New("battery not configured")

const (
	GridTariff    = "grid"
	FeedinTariff  = "feedin"
	PlannerTariff = "planner"
)

// isConfigurable checks if the meter is configurable
func isConfigurable(ref string) bool {
	dev, _ := config.Meters().ByName(ref)
	_, ok := dev.(config.ConfigurableDevice[api.Meter])
	return ok
}

// filterConfigurable filters configurable meters
func filterConfigurable(ref []string) []string {
	var res []string
	for _, r := range ref {
		if isConfigurable(r) {
			res = append(res, r)
		}
	}
	return res
}

// GetTitle returns the title
func (site *Site) GetTitle() string {
	site.RLock()
	defer site.RUnlock()
	return site.Title
}

// SetTitle sets the title
func (site *Site) SetTitle(title string) {
	site.Lock()
	defer site.Unlock()

	site.Title = title
	site.publish(keys.SiteTitle, title)
	settings.SetString(keys.Title, title)
}

// GetGridMeterRef returns the GridMeterRef
func (site *Site) GetGridMeterRef() string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.GridMeterRef
}

// SetGridMeterRef sets the GridMeterRef
func (site *Site) SetGridMeterRef(ref string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.GridMeterRef = ref
	settings.SetString(keys.GridMeter, ref)
}

// GetPVMeterRefs returns the PvMeterRef
func (site *Site) GetPVMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.PVMetersRef
}

// SetPVMeterRefs sets the PvMeterRef
func (site *Site) SetPVMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.PVMetersRef = ref
	settings.SetString(keys.PvMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetBatteryMeterRefs returns the BatteryMeterRef
func (site *Site) GetBatteryMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.BatteryMetersRef
}

// SetBatteryMeterRefs sets the BatteryMeterRef
func (site *Site) SetBatteryMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.BatteryMetersRef = ref
	settings.SetString(keys.BatteryMeters, strings.Join(filterConfigurable(ref), ","))
}

// GetAuxMeterRefs returns the AuxMeterRef
func (site *Site) GetAuxMeterRefs() []string {
	site.RLock()
	defer site.RUnlock()
	return site.Meters.AuxMetersRef
}

// SetAuxMeterRefs sets the AuxMeterRef
func (site *Site) SetAuxMeterRefs(ref []string) {
	site.Lock()
	defer site.Unlock()

	site.Meters.AuxMetersRef = ref
	settings.SetString(keys.AuxMeters, strings.Join(filterConfigurable(ref), ","))
}

// Loadpoints returns the loadpoints as api interfaces
func (site *Site) Loadpoints() []loadpoint.API {
	return lo.Map(site.loadpoints, func(lp *Loadpoint, _ int) loadpoint.API { return lp })
}

// loadpointsAsCircuitDevices returns the loadpoints as circuit devices
func (site *Site) loadpointsAsCircuitDevices() []api.CircuitLoad {
	return lo.Map(site.loadpoints, func(lp *Loadpoint, _ int) api.CircuitLoad { return lp })
}

// Vehicles returns the site vehicles
func (site *Site) Vehicles() site.Vehicles {
	return &vehicles{log: site.log}
}

// GetCircuit returns the root circuit
func (site *Site) GetCircuit() api.Circuit {
	site.RLock()
	defer site.RUnlock()
	return site.circuit
}

// SetCircuit sets the root circuit
func (site *Site) SetCircuit(circuit api.Circuit) {
	site.Lock()
	defer site.Unlock()
	site.circuit = circuit
}

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

	if site.bufferSoc != 0 && soc > site.bufferSoc {
		return errors.New("priority soc must be smaller or equal than buffer soc")
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

	if soc != 0 && soc < site.prioritySoc {
		return errors.New("buffer soc must not be smaller than priority soc")
	}

	if site.bufferStartSoc != 0 && soc > site.bufferStartSoc {
		return errors.New("buffer soc must be smaller or equal than buffer start soc")
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

	if soc != 0 && soc < site.bufferSoc {
		return errors.New("buffer start soc must be larger than buffer soc")
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
	site.log.DEBUG.Println("set residual power:", power)

	site.Lock()
	defer site.Unlock()

	if site.ResidualPower != power {
		site.ResidualPower = power
		settings.SetFloat(keys.ResidualPower, site.ResidualPower)
		site.publish(keys.ResidualPower, site.ResidualPower)
	}

	return nil
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
			// prio 1: grid tariff with forecast
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

// GetBatteryDischargeControl returns the battery control mode (no discharge only)
func (site *Site) GetBatteryDischargeControl() bool {
	site.RLock()
	defer site.RUnlock()
	return site.batteryDischargeControl
}

// SetBatteryDischargeControl sets the battery control mode (no discharge only)
func (site *Site) SetBatteryDischargeControl(val bool) error {
	site.log.DEBUG.Println("set battery discharge control:", val)

	site.Lock()
	defer site.Unlock()

	if site.batteryDischargeControl != val {
		site.batteryDischargeControl = val
		settings.SetBool(keys.BatteryDischargeControl, val)
		site.publish(keys.BatteryDischargeControl, val)
	}

	return nil
}

func (site *Site) GetBatteryGridChargeLimit() *float64 {
	site.RLock()
	defer site.RUnlock()
	return site.batteryGridChargeLimit
}

func (site *Site) SetBatteryGridChargeLimit(val *float64) {
	site.log.DEBUG.Println("set grid charge limit:", printPtr("%.1f", val))

	site.Lock()
	defer site.Unlock()

	if !ptrValueEqual(site.batteryGridChargeLimit, val) {
		site.batteryGridChargeLimit = val

		if val == nil {
			settings.SetString(keys.BatteryGridChargeLimit, "")
			site.publish(keys.BatteryGridChargeLimit, nil)
		} else {
			settings.SetFloat(keys.BatteryGridChargeLimit, *val)
			site.publish(keys.BatteryGridChargeLimit, *val)
		}
	}
}
