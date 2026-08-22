package core

import (
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

func (c *CurrentController) PublishEffectiveValues() {
	c.lp.publish(keys.EffectiveMinCurrent, c.effectiveMinCurrent())
	c.lp.publish(keys.EffectiveMaxCurrent, c.effectiveMaxCurrent())
}

// effectiveMinCurrent returns the effective min current
func (c *CurrentController) effectiveMinCurrent() float64 {
	lpMin := c.minCurrent
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

	// power-limited chargers (e.g. EEBus OHPCF heat pump) report their demand in
	// W; convert to per-phase current so the PV enable gate covers it
	if pl, ok := api.Cap[api.PowerLimiter](c.lp.charger); ok {
		if res, _, err := pl.GetMinMaxPower(); err == nil && res > 0 {
			chargerMin = res / (Voltage * float64(c.lp.minActivePhases()))
			// coarse chargers truncate to full amps in setLimit, so round the
			// demand up to keep the enable gate reachable (#31549)
			if c.coarseCurrent() {
				chargerMin = math.Ceil(chargerMin)
			}
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
	maxCurrent := c.maxCurrent

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

	if pl, ok := api.Cap[api.PowerLimiter](c.lp.charger); ok {
		if _, res, err := pl.GetMinMaxPower(); err == nil && res > 0 {
			powerMax := res / (Voltage * float64(c.lp.maxActivePhases()))
			// match effectiveMinCurrent's rounding so a fixed power request
			// (min == max) doesn't yield min > max on coarse chargers (#31549)
			if c.coarseCurrent() {
				powerMax = math.Ceil(powerMax)
			}
			maxCurrent = min(maxCurrent, powerMax)
		}
	}

	return maxCurrent
}

// MinPower returns the lower bound of the capability envelope in W. With automatic
// phase switching it spans down to the 1p minimum, with locked phase configuration
// it reflects the locked phases.
func (c *CurrentController) MinPower() float64 {
	phases := c.lp.minActivePhases()
	if c.lp.hasPhaseSwitching() && c.lp.phasesConfigured > 1 {
		phases = c.lp.phasesConfigured
	}
	return currentToPower(c.effectiveMinCurrent(), phases)
}

// MaxPower returns the upper bound of the capability envelope in W
func (c *CurrentController) MaxPower() float64 {
	return c.effectiveMaxPower()
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

// stepPower returns the power step of one full amp on the currently active phases
func (c *CurrentController) stepPower() float64 {
	return Voltage * float64(c.lp.ActivePhases())
}

// phaseSwitchGapPower returns the extra power needed to bridge the gap between the
// maximum power on the active phases and the minimum power after scaling up, if
// scaling up is possible
func (c *CurrentController) phaseSwitchGapPower() float64 {
	activePhases, maxPhases := c.lp.ActivePhases(), c.lp.MaxActivePhases()
	if activePhases >= maxPhases || !c.circuitAllowsPhases(maxPhases, c.effectiveMinCurrent()) {
		return 0
	}

	// max power actually achievable on the active phases
	activeMaxPower := min(c.lp.EffectiveMaxPower(), c.activeMaxPower())
	return max(0, c.lp.EffectiveMinPower()*float64(maxPhases)-activeMaxPower)
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
