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

// SetBatteryMode sets the battery mode
func (site *Site) SetBatteryMode(batMode api.BatteryMode) {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set battery mode:", batMode)

	if site.batteryMode != batMode {
		site.batteryMode = batMode
		site.publish(keys.BatteryMode, batMode)
	}
}

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
	site.SetBatteryMode(mode)

	return nil
}

func (site *Site) smartCostActive(lp loadpoint.API) (bool, error) {
	tariff := site.GetTariff(PlannerTariff)
	if tariff == nil || tariff.Type() == api.TariffTypePriceStatic {
		return false, nil
	}

	rates, err := tariff.Rates()
	if err != nil {
		return false, err
	}

	rate, err := rates.Current(time.Now())
	if err != nil {
		return false, err
	}

	limit := lp.GetSmartCostLimit()
	return limit != 0 && rate.Price <= limit, nil
}

func (site *Site) updateBatteryMode() {
	mode := api.BatteryNormal

	for _, lp := range site.Loadpoints() {
		smartCostActive, err := site.smartCostActive(lp)
		if err != nil {
			site.log.ERROR.Println("smart cost:", err)
			continue
		}

		if lp.GetStatus() == api.StatusC && (smartCostActive || lp.IsFastChargingActive()) {
			mode = api.BatteryHold
			break
		}
	}

	if batMode := site.GetBatteryMode(); mode != batMode {
		if err := site.applyBatteryMode(mode); err != nil {
			site.log.ERROR.Println("battery mode:", err)
		}
	}
}
