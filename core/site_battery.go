package core

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/util/config"
)

func batteryModeModified(mode api.BatteryMode) bool {
	return mode != api.BatteryUnknown && mode != api.BatteryNormal
}

func (site *Site) batteryConfigured() bool {
	return len(site.batteryMeters) > 0
}

func (site *Site) hasBatteryControl() bool {
	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		if api.HasCap[api.BatteryController](meter) {
			return true
		}
	}

	return false
}

func (site *Site) hasBatteryChargePowerLimitControl() bool {
	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		if api.HasCap[api.BatteryChargePowerLimiter](meter) {
			return true
		}
	}

	return false
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

	if site.batteryModeExternal == api.BatteryUnknown {
		site.batteryModeExternalTimer = time.Time{}
	}
}

func (site *Site) updateBatteryMode(batteryGridChargeActive bool, rate api.Rate) {
	batteryMode := site.requiredBatteryMode(batteryGridChargeActive, rate)

	// put battery into hold mode when charging is active and HEMS dimmed
	fromToCharge := batteryMode == api.BatteryCharge || batteryMode == api.BatteryUnknown && site.batteryMode == api.BatteryCharge
	if dimmed := hems.Dimmed(site.hems); fromToCharge && dimmed != nil && *dimmed {
		site.log.DEBUG.Println("battery mode: HEMS dimmed")
		batteryMode = api.BatteryHold
	}

	// NOTE: applyBatteryMode is always called when charge mode is active to validate max soc
	if modeChanged := batteryMode != api.BatteryUnknown; modeChanged || site.batteryMode == api.BatteryCharge {
		if err := site.applyBatteryMode(batteryMode); err == nil {
			if modeChanged {
				site.SetBatteryMode(batteryMode)
			}
		} else {
			site.log.ERROR.Println("battery mode:", err)
		}
	}
}

// requiredBatteryMode determines required battery mode based on grid charge and rate
func (site *Site) requiredBatteryMode(batteryGridChargeActive bool, rate api.Rate) api.BatteryMode {
	var res api.BatteryMode
	batMode := site.GetBatteryMode()
	extMode := site.GetBatteryModeExternal()

	var extModeReset bool
	if extMode == api.BatteryUnknown {
		site.Lock()
		extModeReset = !site.batteryModeExternalTimer.IsZero()
		site.Unlock()
	}

	keepUnlessModified := func(s api.BatteryMode) api.BatteryMode {
		return map[bool]api.BatteryMode{false: s, true: api.BatteryUnknown}[batMode == s]
	}

	switch {
	case !site.batteryConfigured():
		res = api.BatteryUnknown
	case extModeReset:
		// require normal mode to leave external control
		res = api.BatteryNormal
	case extMode != api.BatteryUnknown:
		// require external mode only once
		if extMode != batMode {
			res = extMode
		}
	case site.Automatic() && site.unmodelledCharging():
		// the suggestion ignores loads the optimizer cannot model as storage
		res = keepUnlessModified(api.BatteryHold)
	case site.Automatic():
		// optimizer decides, replacing grid charge limit and discharge control
		if mode, ok := site.batterySuggestionMode(); ok {
			res = keepUnlessModified(mode)
		} else if batteryModeModified(batMode) {
			// no suggestion: release the battery
			res = api.BatteryNormal
		}
	case batteryGridChargeActive:
		res = keepUnlessModified(api.BatteryCharge)
	case site.dischargeControlActive(rate):
		res = keepUnlessModified(api.BatteryHold)
	case batteryModeModified(batMode):
		res = api.BatteryNormal
	}

	return res
}

// unmodelledCharging reports a loadpoint charging at full power that the optimizer
// cannot model as storage (unknown vehicle capacity, see optimizerRequest). Its
// battery suggestion does not account for that load, so the battery must be held.
func (site *Site) unmodelledCharging() bool {
	for _, lp := range site.activeLoadpoints() {
		if v := lp.GetVehicle(); v != nil && v.Capacity() > 0 {
			continue
		}

		if lp.GetStatus() == api.StatusC && lp.IsFastChargingActive() {
			return true
		}
	}

	return false
}

// batterySuggestionMode returns the optimizer's mode for the first controllable battery.
// TODO apply per battery once the site tracks more than a single battery mode
func (site *Site) batterySuggestionMode() (api.BatteryMode, bool) {
	for _, dev := range site.batteryMeters {
		if dev == nil {
			continue
		}

		name := dev.Config().Name

		s := site.suggestion(batteryKey(name), site.GetBatteryMode().String())
		if s == nil {
			continue
		}

		mode, err := api.BatteryModeString(s.Action)
		if err != nil {
			// discharging to grid has no matching battery mode
			site.log.DEBUG.Printf("battery %s: cannot apply suggestion %s", name, s.Action)
			return api.BatteryNormal, true
		}

		return mode, true
	}

	return api.BatteryUnknown, false
}

// batteryMaxSocReached checks is battery has exceed max soc limit
func (site *Site) batteryMaxSocReached(dev config.Device[api.Meter]) (bool, error) {
	meter := dev.Instance()

	batLimiter, ok := api.Cap[api.BatterySocLimiter](meter)
	if !ok {
		return false, nil
	}

	batSoc, ok := api.Cap[api.Battery](meter)
	if !ok {
		return false, errors.New("battery with soc limits must have soc")
	}

	soc, err := batSoc.Soc()
	if err != nil {
		return false, err
	}

	if _, max := batLimiter.GetSocLimits(); max > 0 && max < 100 && soc >= max {
		site.log.DEBUG.Printf("battery %s: limit soc reached (%.0f > %.0f)", deviceTitleOrName(dev), soc, max)
		return true, nil
	}

	return false, nil
}

// applyBatteryMode applies the mode to each battery
//
// api.BatteryCharge:
//
//	The current soc is validated against max soc.
//	In case max soc is reached, hold mode is applied to that battery only.
func (site *Site) applyBatteryMode(mode api.BatteryMode) error {
	fromToCharge := mode == api.BatteryCharge || mode == api.BatteryUnknown && site.batteryMode == api.BatteryCharge

	if site.batteryModeApplied == nil {
		site.batteryModeApplied = make(map[string]api.BatteryMode)
	}

	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		batCtrl, ok := api.Cap[api.BatteryController](meter)
		if !ok {
			continue
		}

		// mode is per battery, max soc is validated individually
		devMode := mode

		// validate max soc
		if fromToCharge && devMode != api.BatteryHold {
			ok, err := site.batteryMaxSocReached(dev)
			if err != nil && !errors.Is(err, api.ErrNotAvailable) {
				return err
			}

			// put battery into hold mode when soc limit reached
			if ok {
				devMode = api.BatteryHold
			}
		}

		// don't re-apply the mode the battery is already in
		name := dev.Config().Name
		if devMode == api.BatteryUnknown || devMode == site.batteryModeApplied[name] {
			continue
		}

		if err := batCtrl.SetBatteryMode(devMode); err != nil {
			if !errors.Is(err, api.ErrNotAvailable) {
				return err
			}
			continue
		}

		site.batteryModeApplied[name] = devMode
		site.log.DEBUG.Printf("set battery %s mode: %s", deviceTitleOrName(dev), devMode)
	}

	return nil
}

// batteryChargingPossible reports whether the battery mode allows charging at all.
// Hold and HoldCharge both mean "don't charge", so a positive charge power cap is meaningless there.
func batteryChargingPossible(mode api.BatteryMode) bool {
	return mode != api.BatteryHold && mode != api.BatteryHoldCharge
}

// setBatteryChargePowerLimit sets the applied battery charge power limit
func (site *Site) setBatteryChargePowerLimit(limit *float64) {
	site.batteryChargePowerLimit = limit
	site.publish(keys.BatteryChargePowerLimit, limit)
}

// applyBatteryChargePowerLimit applies the charge power limit to each battery
func (site *Site) applyBatteryChargePowerLimit(limit *float64) error {
	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		limiter, ok := api.Cap[api.BatteryChargePowerLimiter](meter)
		if !ok {
			continue
		}

		if err := limiter.SetBatteryChargePowerLimit(limit); err != nil && !errors.Is(err, api.ErrNotAvailable) {
			return err
		}
	}

	return nil
}

// updateBatteryChargePowerLimit applies the external battery charge power limit, scoped to modes
// where charging is possible, and avoids redundant device writes when nothing has changed
func (site *Site) updateBatteryChargePowerLimit() {
	site.Lock()
	var required *float64
	if ext := site.batteryChargePowerLimitExternal; ext != nil && batteryChargingPossible(site.batteryMode) {
		required = ext
	}
	changed := !ptrValueEqual(site.batteryChargePowerLimit, required)
	site.Unlock()

	if !changed {
		return
	}

	if err := site.applyBatteryChargePowerLimit(required); err != nil {
		site.log.ERROR.Println("battery charge power limit:", err)
		return
	}

	site.Lock()
	site.setBatteryChargePowerLimit(required)
	site.Unlock()
}

func (site *Site) tariffRates(usage api.TariffUsage) (api.Rates, error) {
	tariff := site.GetTariff(usage)
	if tariff == nil || tariff.Type() == api.TariffTypePriceStatic {
		return nil, nil
	}

	return tariff.Rates()
}

func (site *Site) smartCostActive(lp loadpoint.API, rate api.Rate) bool {
	limit := lp.GetSmartCostLimit()
	return limit != nil && !rate.IsZero() && rate.Value <= *limit
}

func (site *Site) batteryGridChargeActive(rate api.Rate) bool {
	limit := site.GetBatteryGridChargeLimit()
	return limit != nil && !rate.IsZero() && rate.Value <= *limit
}

func (site *Site) dischargeControlActive(rate api.Rate) bool {
	if !site.GetBatteryDischargeControl() {
		return false
	}

	for _, lp := range site.activeLoadpoints() {
		smartCostActive := site.smartCostActive(lp, rate)
		if lp.GetStatus() == api.StatusC && (smartCostActive || lp.IsFastChargingActive()) {
			return true
		}
	}

	return false
}
