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

// applyBatteryMode applies the mode to each battery
func (site *Site) applyBatteryMode(mode api.BatteryMode) error {
	for _, meter := range site.batteryMeters {
		if batCtrl, ok := meter.(api.BatteryController); ok {
			if err := batCtrl.SetBatteryMode(mode); err != nil {
				return err
			}
		}
	}

	return nil
}

func (site *Site) plannerRates() (api.Rates, error) {
	tariff := site.GetTariff(PlannerTariff)
	if tariff == nil || tariff.Type() == api.TariffTypePriceStatic {
		return nil, nil
	}

	return tariff.Rates()
}

func (site *Site) smartCostActive(lp loadpoint.API, rate api.Rate) bool {
	limit := lp.GetSmartCostLimit()
	return limit != nil && !rate.IsEmpty() && rate.Price <= *limit
}

func (site *Site) smartCostNextStart(lp loadpoint.API, rates api.Rates) time.Time {
	limit := lp.GetSmartCostLimit()
	if limit == nil || rates == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rates {
		if slot.Start.After(now) && slot.Price <= *limit {
			return slot.Start
		}
	}

	return time.Time{}
}

func (site *Site) gridChargeActive(rate api.Rate) bool {
	limit := site.GetGridChargeLimit()
	return limit != nil && !rate.IsEmpty() && rate.Price <= *limit
}

func (site *Site) dischargeControlActive(rate api.Rate) bool {
	for _, lp := range site.Loadpoints() {
		smartCostActive := site.smartCostActive(lp, rate)
		if lp.GetStatus() == api.StatusC && (smartCostActive || lp.IsFastChargingActive()) {
			return true
		}
	}

	return false
}
