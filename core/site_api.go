package core

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
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

// Loadpoints returns the list loadpoints
func (site *Site) Loadpoints() []loadpoint.API {
	res := make([]loadpoint.API, len(site.loadpoints))
	for id, lp := range site.loadpoints {
		res[id] = lp
	}
	return res
}

// Vehicles returns the site vehicles
func (site *Site) Vehicles() site.Vehicles {
	return &vehicles{log: site.log}
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

	if soc != 0 && soc <= site.prioritySoc {
		return errors.New("buffer soc must be larger than priority soc")
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

	if soc != 0 && soc <= site.bufferSoc {
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
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set residual power:", power)

	if site.ResidualPower != power {
		site.ResidualPower = power
		site.publish(keys.ResidualPower, site.ResidualPower)
	}

	return nil
}

// GetSmartCostLimit returns the smartCostLimit
func (site *Site) GetSmartCostLimit() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.smartCostLimit
}

// SetSmartCostLimit sets the smartCostLimit
func (site *Site) SetSmartCostLimit(val float64) error {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set smart cost limit:", val)

	if site.smartCostLimit != val {
		site.smartCostLimit = val
		settings.SetFloat(keys.SmartCostLimit, site.smartCostLimit)
		site.publish(keys.SmartCostLimit, site.smartCostLimit)
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

// GetBatteryControl returns the battery control mode
func (site *Site) GetBatteryDischargeControl() bool {
	site.RLock()
	defer site.RUnlock()
	return site.batteryDischargeControl
}

// SetBatteryControl sets the battery control mode
func (site *Site) SetBatteryDischargeControl(val bool) error {
	site.log.DEBUG.Println("set battery discharge control:", val)

	if site.GetBatteryDischargeControl() != val {
		// reset to normal when disabling
		if mode := site.GetBatteryMode(); mode != api.BatteryNormal {
			if err := site.updateBatteryMode(api.BatteryNormal); err != nil {
				return err
			}
		}

		site.Lock()
		defer site.Unlock()

		site.batteryDischargeControl = val
		settings.SetBool(keys.BatteryDischargeControl, val)
		site.publish(keys.BatteryDischargeControl, val)
	}

	return nil
}
