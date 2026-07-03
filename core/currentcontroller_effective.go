package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

func (c *CurrentController) PublishEffectiveValues() {
	c.lp.publish(keys.EffectiveMinCurrent, c.effectiveMinCurrent())
	c.lp.publish(keys.EffectiveMaxCurrent, c.effectiveMaxCurrent())
}

// effectiveMinCurrent returns the effective min current
func (c *CurrentController) effectiveMinCurrent() float64 {
	lpMin := c.lp.getMinCurrent()
	var vehicleMin, chargerMin float64

	if v := c.lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			vehicleMin = res
		}
	}

	if cl, ok := api.Cap[api.CurrentLimiter](c.lp.charger); ok {
		if res, _, err := cl.GetMinMaxCurrent(); err == nil {
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
func (c *CurrentController) effectiveMaxCurrent() float64 {
	maxCurrent := c.lp.getMaxCurrent()

	if v := c.lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	if cl, ok := api.Cap[api.CurrentLimiter](c.lp.charger); ok {
		if _, res, err := cl.GetMinMaxCurrent(); err == nil && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	return maxCurrent
}

// effectiveMinPower returns the effective min power for the minimum active phases
func (c *CurrentController) effectiveMinPower() float64 {
	return Voltage * c.effectiveMinCurrent() * float64(c.lp.minActivePhases())
}

// activeMinPower returns the min power at the currently active phases
func (c *CurrentController) activeMinPower() float64 {
	return currentToPower(c.effectiveMinCurrent(), c.lp.ActivePhases())
}

// activeMaxPower returns the max power at the currently active phases
func (c *CurrentController) activeMaxPower() float64 {
	return currentToPower(c.effectiveMaxCurrent(), c.lp.ActivePhases())
}

// reachableMinPower returns the min power taking an immediate or pending
// phase scale-down into account
func (c *CurrentController) reachableMinPower() float64 {
	phases := c.lp.ActivePhases()
	if c.lp.hasPhaseSwitching() && c.lp.phasesConfigured < 3 && phases > 1 {
		phases = 1
	}
	return currentToPower(c.effectiveMinCurrent(), phases)
}

// effectivePower returns the currently effective charging power
func (c *CurrentController) effectivePower() float64 {
	// for slow-acting heating devices, only take actually consumed power into account
	if c.lp.chargerHasFeature(api.IntegratedDevice) {
		return c.lp.chargePower
	}
	return currentToPower(c.effectiveCurrent(), c.lp.ActivePhases())
}

// effectiveMaxPower returns the effective max power taking vehicle capabilities and phase scaling into account
func (c *CurrentController) effectiveMaxPower() float64 {
	res := Voltage * c.effectiveMaxCurrent() * float64(c.lp.maxActivePhases())
	if c.lp.vehicle != nil {
		if maxPower, ok := c.lp.vehicle.OnIdentified().GetMaxPower(); ok {
			return min(maxPower, res)
		}
	}
	return res
}
