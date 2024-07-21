package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
)

func batteryModeModified(mode api.BatteryMode) bool {
	return mode != api.BatteryUnknown && mode != api.BatteryNormal
}

// GetBatteryMode returns the battery mode
func (site *Site) GetBatteryMode() api.BatteryMode {
	site.RLock()
	defer site.RUnlock()
	return site.batteryMode
}

// setBatteryMode sets the battery mode
func (site *Site) setBatteryMode(batMode api.BatteryMode) {
	site.batteryMode = batMode
	site.publish(keys.BatteryMode, batMode)
}

// SetBatteryMode sets the battery mode
func (site *Site) SetBatteryMode(batMode api.BatteryMode) {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set battery mode:", batMode)

	if site.batteryMode != batMode {
		site.setBatteryMode(batMode)
	}
}

// applyBatteryMode applies the mode to each battery and updates
// internal state if successful (requires lock)
func (site *Site) applyBatteryMode(mode api.BatteryMode) error {
	// update batteries
	for _, meter := range site.batteryMeters {
		if batCtrl, ok := meter.(api.BatteryController); ok {
			if err := batCtrl.SetBatteryMode(mode); err != nil {
				return err
			}
		}
	}

	// update state and publish
	site.setBatteryMode(mode)

	return nil
}

func (site *Site) plannerRates() (api.Rates, error) {
	tariff := site.GetTariff(PlannerTariff)
	if tariff == nil || tariff.Type() == api.TariffTypePriceStatic {
		return nil, nil
	}

	return tariff.Rates()
}

func (site *Site) plannerRate() (*api.Rate, error) {
	rates, err := site.plannerRates()
	if rates == nil || err != nil {
		return nil, err
	}

	rate, err := rates.Current(time.Now())
	if err != nil {
		return nil, err
	}

	return &rate, nil
}

func (site *Site) smartCostActive(lp loadpoint.API, rate *api.Rate) bool {
	limit := lp.GetSmartCostLimit()
	return limit != 0 && rate != nil && rate.Price <= limit
}

func (site *Site) smartCostNextStart(lp loadpoint.API, rate api.Rates) time.Time {
	limit := lp.GetSmartCostLimit()
	if limit == 0 || rate == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rate {
		if slot.Start.After(now) && slot.Price <= limit {
			return slot.Start
		}
	}

	return time.Time{}
}

func (site *Site) updateBatteryMode() {
	mode := api.BatteryNormal

	rate, err := site.plannerRate()
	if err != nil {
		site.log.WARN.Println("smart cost:", err)
	}

	for _, lp := range site.Loadpoints() {
		smartCostActive := site.smartCostActive(lp, rate)
		if lp.GetStatus() == api.StatusC && (smartCostActive || lp.IsFastChargingActive()) {
			mode = api.BatteryHold
			break
		}
	}

	if batMode := site.GetBatteryMode(); mode != batMode {
		site.Lock()
		if err := site.applyBatteryMode(mode); err != nil {
			site.log.ERROR.Println("battery mode:", err)
		}
		site.Unlock()
	}
}
