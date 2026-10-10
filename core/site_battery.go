package core

import (
	"errors"
	"slices"
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

// batteryModesModified reports whether the site or any single battery is in a modified mode
func (site *Site) batteryModesModified() bool {
	if batteryModeModified(site.GetBatteryMode()) {
		return true
	}

	for _, mode := range site.batteryModeApplied {
		if batteryModeModified(mode) {
			return true
		}
	}

	return false
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

// fromTo reports whether the battery is entering, holding or leaving mode m:
// either m is requested, or nothing is requested and the battery is already in m.
func (site *Site) fromTo(requested, m api.BatteryMode) bool {
	return requested == m || requested == api.BatteryUnknown && site.batteryMode == m
}

// fromToAny reports fromTo for the site mode or any of the per-battery modes
func (site *Site) fromToAny(requested api.BatteryMode, modes map[string]api.BatteryMode, m api.BatteryMode) bool {
	for _, mode := range modes {
		if mode == m {
			return true
		}
	}

	return site.fromTo(requested, m)
}

func (site *Site) updateBatteryMode(batteryGridChargeActive, batteryGridDischargeActive bool, rate api.Rate) {
	batteryMode, deviceModes := site.requiredBatteryMode(batteryGridChargeActive, batteryGridDischargeActive, rate)

	// put battery into hold mode when charging is active and HEMS dimmed; HEMS overrides apply to every battery
	if dimmed := hems.Dimmed(site.hems); site.fromToAny(batteryMode, deviceModes, api.BatteryCharge) && dimmed != nil && *dimmed {
		site.log.DEBUG.Println("battery mode: HEMS dimmed")
		batteryMode, deviceModes = api.BatteryHold, nil
	}

	// stop discharging to grid when HEMS curtailed production, but keep self-consumption
	if curtailed := hems.Curtailed(site.hems); site.fromToAny(batteryMode, deviceModes, api.BatteryDischarge) && curtailed != nil && *curtailed {
		site.log.DEBUG.Println("battery mode: HEMS curtailed")
		batteryMode, deviceModes = api.BatteryNormal, nil
	}

	// NOTE: applyBatteryMode is always called when charge or discharge mode is active to
	// validate max soc / min soc reserve, and whenever batteries follow their own modes
	if modeChanged := batteryMode != api.BatteryUnknown; modeChanged || site.batteryMode == api.BatteryCharge || site.batteryMode == api.BatteryDischarge || len(deviceModes) > 0 {
		if err := site.applyBatteryMode(batteryMode, deviceModes); err == nil {
			if modeChanged {
				site.SetBatteryMode(batteryMode)
			}
		} else {
			site.log.ERROR.Println("battery mode:", err)
		}
	}
}

// requiredBatteryMode determines the required battery mode based on grid charge/discharge and rate.
// In automatic mode every battery follows its own optimizer suggestion, returned by battery name.
func (site *Site) requiredBatteryMode(batteryGridChargeActive, batteryGridDischargeActive bool, rate api.Rate) (api.BatteryMode, map[string]api.BatteryMode) {
	var res api.BatteryMode
	var modes map[string]api.BatteryMode
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
		// optimizer decides per battery, replacing grid charge limit and discharge control;
		// the site mode follows the first battery with a suggestion
		var first api.BatteryMode
		modes, first = site.batterySuggestionModes()

		if first != api.BatteryUnknown {
			res = keepUnlessModified(first)
		} else if site.batteryModesModified() {
			// no suggestion: release the batteries
			res = api.BatteryNormal
		}
	case batteryGridChargeActive:
		// independent limits (buy vs feed-in rate) can both be active at once;
		// charge wins to avoid buying and immediately selling
		if batteryGridDischargeActive && batMode != api.BatteryCharge {
			site.log.WARN.Println("battery mode: grid charge and grid discharge both active, charge takes priority")
		}
		res = keepUnlessModified(api.BatteryCharge)
	case site.dischargeControlActive(rate) || (batteryGridDischargeActive && site.evFastChargingActive()):
		// hold wins over feed-in discharge; fast charging holds even without
		// batteryDischargeControl, selling while an EV fast-charges is worse
		res = keepUnlessModified(api.BatteryHold)
	case batteryGridDischargeActive:
		res = keepUnlessModified(api.BatteryDischarge)
	case batteryModeModified(batMode):
		res = api.BatteryNormal
	}

	return res, modes
}

// unmodelledCharging reports a loadpoint charging at full power that the optimizer
// does not control: it cannot model it as storage (unknown vehicle capacity, see
// optimizerRequest) or only the battery is automatic. Its battery suggestion does
// not account for that load, so the battery must be held.
func (site *Site) unmodelledCharging() bool {
	for _, lp := range site.activeLoadpoints() {
		if lp.optimizerControlled() {
			continue
		}

		if lp.GetStatus() == api.StatusC && lp.IsFastChargingActive() {
			return true
		}
	}

	return false
}

// batterySuggestionModes returns the optimizer's mode per battery with a suggestion
// and the mode of the first of them, BatteryUnknown if there is none
func (site *Site) batterySuggestionModes() (map[string]api.BatteryMode, api.BatteryMode) {
	modes := make(map[string]api.BatteryMode)
	first := api.BatteryUnknown

	for _, dev := range site.batteryMeters {
		if dev == nil {
			continue
		}

		name := dev.Config().Name

		s := site.suggestion(batteryKey(name), site.batteryModeApplied[name].String())
		if s == nil {
			continue
		}

		mode, err := api.BatteryModeString(s.Action)
		if err != nil {
			// unknown action, release the battery
			site.log.DEBUG.Printf("battery %s: cannot apply suggestion %s", deviceTitleOrName(dev), s.Action)
			mode = api.BatteryNormal
		}

		modes[name] = mode
		if first == api.BatteryUnknown {
			first = mode
		}
	}

	return modes, first
}

// batterySocLimitReached reports whether the battery has reached the soc bound
// that should stop the requested mode: the max soc when charging, or the min
// soc reserve when discharging to grid. A configured limit of 0 disables the
// respective check (max is also disabled at 100).
func (site *Site) batterySocLimitReached(dev config.Device[api.Meter], discharge bool) (bool, error) {
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

	minSoc, maxSoc := batLimiter.GetSocLimits()

	if discharge {
		if minSoc > 0 && soc <= minSoc {
			site.log.DEBUG.Printf("battery %s: reserve soc reached (%.0f <= %.0f)", deviceTitleOrName(dev), soc, minSoc)
			return true, nil
		}
		return false, nil
	}

	if maxSoc > 0 && maxSoc < 100 && soc >= maxSoc {
		site.log.DEBUG.Printf("battery %s: limit soc reached (%.0f >= %.0f)", deviceTitleOrName(dev), soc, maxSoc)
		return true, nil
	}

	return false, nil
}

// batteryModeFallbacks lists the modes applied instead of an unsupported mode, most to least aligned with its intent
var batteryModeFallbacks = map[api.BatteryMode][]api.BatteryMode{
	api.BatteryCharge:     {api.BatteryHold, api.BatteryNormal},       // keep the energy while grid power is cheap
	api.BatteryDischarge:  {api.BatteryHoldCharge, api.BatteryNormal}, // don't store pv, feed the house instead
	api.BatteryHold:       {api.BatteryNormal},
	api.BatteryHoldCharge: {api.BatteryNormal},
}

// supportedBatteryMode returns the requested mode if supported, otherwise the first supported fallback or unknown
func supportedBatteryMode(supported []api.BatteryMode, mode api.BatteryMode) api.BatteryMode {
	for _, m := range append([]api.BatteryMode{mode}, batteryModeFallbacks[mode]...) {
		if slices.Contains(supported, m) {
			return m
		}
	}
	return api.BatteryUnknown
}

// applyBatteryMode applies the mode to each battery. Batteries listed in modes follow
// their own mode, the optimizer suggestion in automatic mode, the others the site mode.
//
// A battery that reached the soc bound of the requested mode is held instead:
// the max soc when charging, the min soc reserve when discharging to grid. This
// is decided per device, so one battery reaching its bound does not force the
// others into hold.
func (site *Site) applyBatteryMode(mode api.BatteryMode, modes map[string]api.BatteryMode) error {
	if site.batteryModeApplied == nil {
		site.batteryModeApplied = make(map[string]api.BatteryMode)
	}

	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		batCtrl, ok := api.Cap[api.BatteryController](meter)
		if !ok {
			continue
		}

		name := dev.Config().Name

		// per-device mode so one battery reaching its soc bound does not affect the others
		deviceMode := mode
		if m, ok := modes[name]; ok {
			deviceMode = m
		}

		// hold at the soc bound of the requested mode (max soc for charge, min soc reserve for grid discharge)
		fromToCharge := site.fromTo(deviceMode, api.BatteryCharge)
		fromToDischarge := site.fromTo(deviceMode, api.BatteryDischarge)

		if (fromToCharge || fromToDischarge) && deviceMode != api.BatteryHold {
			hold, err := site.batterySocLimitReached(dev, fromToDischarge)
			if err != nil && !errors.Is(err, api.ErrNotAvailable) {
				return err
			}
			if hold {
				deviceMode = api.BatteryHold
			}
		}

		if deviceMode == api.BatteryUnknown {
			continue
		}

		// an unsupported mode falls back to the closest supported one instead of leaving the battery in the mode applied before
		applyMode := supportedBatteryMode(batCtrl.BatteryModes(), deviceMode)
		if applyMode == api.BatteryUnknown {
			continue
		}

		// don't re-apply the mode the battery is already in
		if applyMode == site.batteryModeApplied[name] {
			continue
		}

		if applyMode != deviceMode {
			site.log.WARN.Printf("battery %s does not support mode %s, using %s", deviceTitleOrName(dev), deviceMode, applyMode)
		}
		deviceMode = applyMode

		if err := batCtrl.SetBatteryMode(deviceMode); err != nil {
			if !errors.Is(err, api.ErrNotAvailable) {
				return err
			}
			continue
		}

		site.batteryModeApplied[name] = deviceMode
		site.log.DEBUG.Printf("set battery %s mode: %s", deviceTitleOrName(dev), deviceMode)
	}

	return nil
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

// batteryGridDischargeActive reports whether the feed-in rate has reached the
// grid discharge limit; the opt-in gates both this and the optimizer's planning
func (site *Site) batteryGridDischargeActive(rate api.Rate) bool {
	if !site.GetBatteryGridDischarge() {
		return false
	}

	limit := site.GetBatteryGridDischargeLimit()
	return limit != nil && !rate.IsZero() && rate.Value >= *limit
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

// evFastChargingActive reports whether any loadpoint is fast charging,
// regardless of the batteryDischargeControl opt-in
func (site *Site) evFastChargingActive() bool {
	for _, lp := range site.activeLoadpoints() {
		if lp.GetStatus() == api.StatusC && lp.IsFastChargingActive() {
			return true
		}
	}

	return false
}
