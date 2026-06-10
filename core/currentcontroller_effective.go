package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

func (lp *CurrentController) PublishEffectiveValues() {
	lp.publish(keys.EffectiveMinCurrent, lp.effectiveMinCurrent())
	lp.publish(keys.EffectiveMaxCurrent, lp.effectiveMaxCurrent())
}

// effectiveMinCurrent returns the effective min current
func (lp *CurrentController) effectiveMinCurrent() float64 {
	lpMin := lp.getMinCurrent()
	var vehicleMin, chargerMin float64

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			vehicleMin = res
		}
	}

	if c, ok := api.Cap[api.CurrentLimiter](lp.charger); ok {
		if res, _, err := c.GetMinMaxCurrent(); err == nil {
			chargerMin = res
		}
	}

	switch {
	case max(vehicleMin, chargerMin) == 0:
		return lpMin
	case chargerMin > 0:
		return max(vehicleMin, chargerMin)
	default:
		return max(vehicleMin, lpMin)
	}
}

// effectiveMaxCurrent returns the effective max current
func (lp *CurrentController) effectiveMaxCurrent() float64 {
	maxCurrent := lp.getMaxCurrent()

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	if c, ok := api.Cap[api.CurrentLimiter](lp.charger); ok {
		if _, res, err := c.GetMinMaxCurrent(); err == nil && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	return maxCurrent
}

// effectiveMinPower returns the effective min power for the minimum active phases
func (lp *CurrentController) effectiveMinPower() float64 {
	return Voltage * lp.effectiveMinCurrent() * float64(lp.minActivePhases())
}

// activeMinPower returns the min power at the currently active phases
func (lp *CurrentController) activeMinPower() float64 {
	return currentToPower(lp.effectiveMinCurrent(), lp.ActivePhases())
}

// activeMaxPower returns the max power at the currently active phases
func (lp *CurrentController) activeMaxPower() float64 {
	return currentToPower(lp.effectiveMaxCurrent(), lp.ActivePhases())
}

// reachableMinPower returns the min power taking an immediate or pending
// phase scale-down into account
func (lp *CurrentController) reachableMinPower() float64 {
	phases := lp.ActivePhases()
	if lp.hasPhaseSwitching() && lp.phasesConfigured < 3 && phases > 1 {
		phases = 1
	}
	return currentToPower(lp.effectiveMinCurrent(), phases)
}

// effectivePower returns the currently effective charging power
func (lp *CurrentController) effectivePower() float64 {
	// for slow-acting heating devices, only take actually consumed power into account
	if lp.chargerHasFeature(api.IntegratedDevice) {
		return lp.chargePower
	}
	return currentToPower(lp.effectiveCurrent(), lp.ActivePhases())
}

// effectiveMaxPower returns the effective max power taking vehicle capabilities and phase scaling into account
func (lp *CurrentController) effectiveMaxPower() float64 {
	res := Voltage * lp.effectiveMaxCurrent() * float64(lp.maxActivePhases())
	if lp.vehicle != nil {
		if maxPower, ok := lp.vehicle.OnIdentified().GetMaxPower(); ok {
			return min(maxPower, res)
		}
	}
	return res
}
