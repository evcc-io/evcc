package core

import "math"

func (lp *Loadpoint) usesCloudBoost() bool {
	return lp.activeSuggestion() == nil && (lp.coordinatedPV() || lp.GetBatteryBoost() != boostDisabled) && (!lp.hasPhaseSwitching() || lp.phasesConfigured != 0)
}

func (lp *Loadpoint) coordinatedPV() bool {
	s, ok := lp.site.(interface{ CoordinatedPVBattery() bool })
	return ok && s.CoordinatedPVBattery() && (!lp.hasPhaseSwitching() || lp.phasesConfigured != 0)
}

// pvCloudBoostCurrent keeps fixed-phase charging within a PV + battery budget.
// sitePower includes the current EV load and adds measured battery discharge
// back to grid power. Subtracting it from measured EV power therefore gives
// PV minus all other consumption and reserve, independent of actual discharge.
func (lp *Loadpoint) pvCloudBoostCurrent(sitePower float64) float64 {
	minCurrent, maxCurrent := lp.effectiveMinCurrent(), lp.effectiveMaxCurrent()
	phases := lp.ActivePhases()
	minimum := currentToPower(minCurrent, phases)
	if !finiteBatteryPVValue(sitePower) || !finiteBatteryPVValue(lp.chargePower) || minimum <= 0 {
		lp.resetPVTimer()
		return 0
	}

	// A negative residual setting must not authorize planned grid imports
	// in this PV-only start / PV+battery continuation policy.
	pvBudget := lp.chargePower - sitePower + min(0, lp.site.GetResidualPower())
	running := lp.enabled && lp.charging()
	var support float64
	soc := lp.site.GetBatterySoc()
	limit := float64(lp.GetBatteryBoostLimit())
	coordinated := lp.coordinatedPV()
	if coordinated {
		limit = lp.site.GetBufferSoc()
	}
	if !coordinated && lp.GetBatteryBoost() != boostHold && finiteBatteryPVValue(soc) && soc <= limit {
		lp.log.DEBUG.Printf("battery boost hold: soc %.1f%% <= %.1f%%", soc, limit)
		lp.setBatteryBoost(boostHold)
	}
	bufferEnabled := !coordinated || (limit > 0 && limit < 100)
	if running && bufferEnabled && (coordinated || lp.GetBatteryBoost() != boostHold) && finiteBatteryPVValue(limit) && finiteBatteryPVValue(soc) && soc > limit {
		if power := lp.site.GetBatteryMaxDischargePower(); power != nil && finiteBatteryPVValue(*power) {
			support = math.Max(0, *power)
		}
		if !coordinated {
			lp.setBatteryBoost(boostContinue)
		}
	}

	// Do not clamp pvBudget before adding support: a negative balance is the
	// house's unmet demand and must consume battery capacity before the EV does.
	budget := pvBudget + support
	lp.log.DEBUG.Printf("cloud boost: pv budget=%.0fW battery limit=%.0fW total=%.0fW minimum=%.0fW running=%t bufferPolicy=%t soc=%.1f%% socLimit=%.1f%%", pvBudget, support, budget, minimum, running, coordinated, soc, limit)
	if budget < minimum {
		// No disable delay or minimum-current override may plan grid imports.
		lp.resetPVTimer()
		return 0
	}
	if !lp.enabled {
		// support is zero before charging, so the start gate is PV-only.
		if lp.pvTimer.IsZero() {
			lp.pvTimer = lp.clock.Now()
		}
		lp.publishTimer(pvTimer, lp.GetEnableDelay(), pvEnable)
		if lp.clock.Since(lp.pvTimer) < lp.GetEnableDelay() {
			return 0
		}
		lp.resetPVTimer()
		return minCurrent
	}
	lp.resetPVTimer()
	return min(maxCurrent, powerToCurrent(budget, phases))
}
