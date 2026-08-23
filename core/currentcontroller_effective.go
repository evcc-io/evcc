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
			chargerMin = res / (Voltage * float64(c.minActivePhases()))
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
			powerMax := res / (Voltage * float64(c.maxActivePhases()))
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

// Envelope is a per-cycle snapshot of the charge controller's power capabilities
// and state, consumed by the loadpoint's power-domain policy
type Envelope struct {
	ActiveMin    float64 // min power at the currently active phases in W
	ActiveMax    float64 // max power at the currently active phases in W
	ReachableMin float64 // min power reachable from the current phase state in W
	EffectiveMin float64 // min power at the minimum active phases in W
	Effective    float64 // currently effective charging power in W
	Step         float64 // power of one full amp step at the active phases in W
	Coarse       bool    // charger or vehicle require full amp steps
	Enabled      bool    // charger enabled state
	ActivePhases int     // currently active phases
}

// Envelope returns the per-cycle snapshot of the controller's power capabilities and state
func (c *CurrentController) Envelope() Envelope {
	return Envelope{
		ActiveMin:    c.activeMinPower(),
		ActiveMax:    c.activeMaxPower(),
		ReachableMin: c.reachableMinPower(),
		EffectiveMin: c.effectiveMinPower(),
		Effective:    c.effectivePower(),
		Step:         c.stepPower(),
		Coarse:       c.coarseCurrent(),
		Enabled:      c.enabled,
		ActivePhases: c.ActivePhases(),
	}
}

// phaseScalePending reports whether a phase scale timer is currently running
func (c *CurrentController) phaseScalePending() bool {
	return c.hasPhaseSwitching() && !c.phaseTimer.IsZero()
}

// MinPower returns the lower bound of the capability envelope in W. With automatic
// phase switching it spans down to the 1p minimum, with locked phase configuration
// it reflects the locked phases.
func (c *CurrentController) MinPower() float64 {
	phases := c.minActivePhases()
	if c.hasPhaseSwitching() && c.phasesConfigured > 1 {
		phases = c.phasesConfigured
	}
	return currentToPower(c.effectiveMinCurrent(), phases)
}

// MaxPower returns the upper bound of the capability envelope in W
func (c *CurrentController) MaxPower() float64 {
	return c.effectiveMaxPower()
}

// effectiveMinPower returns the effective min power for the minimum active phases
func (c *CurrentController) effectiveMinPower() float64 {
	return Voltage * c.effectiveMinCurrent() * float64(c.minActivePhases())
}

// activeMinPower returns the min power at the currently active phases
func (c *CurrentController) activeMinPower() float64 {
	return currentToPower(c.effectiveMinCurrent(), c.ActivePhases())
}

// activeMaxPower returns the max power at the currently active phases
func (c *CurrentController) activeMaxPower() float64 {
	return currentToPower(c.effectiveMaxCurrent(), c.ActivePhases())
}

// reachableMinPower returns the min power taking an immediate or pending
// phase scale-down into account
func (c *CurrentController) reachableMinPower() float64 {
	phases := c.ActivePhases()
	if c.hasPhaseSwitching() && c.phasesConfigured < 3 && phases > 1 {
		phases = 1
	}
	return currentToPower(c.effectiveMinCurrent(), phases)
}

// stepPower returns the power step of one full amp on the currently active phases
func (c *CurrentController) stepPower() float64 {
	return Voltage * float64(c.ActivePhases())
}

// phaseSwitchGapPower returns the extra power needed to bridge the gap between the
// maximum power on the active phases and the minimum power after scaling up, if
// scaling up is possible
func (c *CurrentController) phaseSwitchGapPower() float64 {
	if !c.hasPhaseSwitching() || !c.phaseSwitchCompleted() {
		return 0
	}

	activePhases, maxPhases := c.ActivePhases(), c.MaxActivePhases()
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
	return currentToPower(c.effectiveCurrent(), c.ActivePhases())
}

// effectiveMaxPower returns the effective max power taking vehicle capabilities and phase scaling into account
func (c *CurrentController) effectiveMaxPower() float64 {
	res := Voltage * c.effectiveMaxCurrent() * float64(c.maxActivePhases())
	if c.lp.vehicle != nil {
		if maxPower, ok := c.lp.vehicle.OnIdentified().GetMaxPower(); ok {
			return min(maxPower, res)
		}
	}
	return res
}
