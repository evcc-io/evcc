package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
)

// boostPower returns the additional power that the loadpoint should draw from the battery
func (lp *Loadpoint) boostPower(batteryBoostPower float64) float64 {
	boost := lp.GetBatteryBoost()
	if boost == boostDisabled {
		return 0
	}

	// push demand to drain battery (at least 100W)
	delta := math.Max(100, math.Abs(lp.site.GetResidualPower()))

	if lp.coarseCurrent() {
		// add effective step power to delta to make sure to step up to the next full amp
		// just using lp.EffectiveStepPower() as delta is not enough because this will result
		// in a too low current when there is a bit remaining grid consumption due to the accuracy
		// of the battery controller
		delta += lp.EffectiveStepPower()
	}

	// start boosting by setting maximum power
	if boost == boostStart {
		delta = lp.EffectiveMaxPower()

		// expire timers
		if lp.hasPhaseSwitching() {
			lp.phaseTimer = elapsed
		}
		lp.pvTimer = elapsed

		if lp.charging() {
			lp.setBatteryBoost(boostContinue)
		}
	}

	res := batteryBoostPower + delta + lp.site.GetResidualPower()
	lp.log.DEBUG.Printf("pv charge battery boost: %.0fW = -%.0fW battery - %.0fW boost", -res, batteryBoostPower, delta)

	return res
}

// pvTargetPower calculates the target charging power for PV mode (0 disables)
func (lp *Loadpoint) pvTargetPower(ctrl *CurrentController, mode api.ChargeMode, sitePower, batteryBoostPower float64, batteryBuffered, batteryStart bool) float64 {
	// read only once to simplify testing
	minPower := ctrl.activeMinPower()
	maxPower := ctrl.activeMaxPower()
	reachableMinPower := ctrl.reachableMinPower()

	// push demand to drain battery
	sitePower -= lp.boostPower(batteryBoostPower)

	// provide surplus for phase reconciliation by the controller
	lp.surplus = &sitePower

	// calculate target charge power from delta power and actual power
	targetPower := max(ctrl.effectivePower()-sitePower, 0)

	// in MinPV mode or under special conditions return at least min power
	if battery := batteryStart || batteryBuffered && lp.charging(); (mode == api.ModeMinPV || battery) && targetPower < minPower {
		lp.log.DEBUG.Printf("pv charge power: min %.0fW > %.0fW (%.0fW @ %dp, battery: %t)", minPower, targetPower, sitePower, lp.ActivePhases(), battery)
		return reachableMinPower
	}

	lp.log.DEBUG.Printf("pv charge power: %.0fW = %.0fW - %.0fW (@ %dp)", targetPower, ctrl.effectivePower(), sitePower, lp.ActivePhases())

	if mode == api.ModePV && lp.enabled && targetPower < minPower {
		projectedSitePower := sitePower
		if lp.hasPhaseSwitching() && !lp.phaseTimer.IsZero() {
			// calculate site power after a phase switch to the minimum reachable phases
			// notes: phaseTimer can only be active if lp current is already at minCurrent
			projectedSitePower -= minPower - reachableMinPower
		}
		// kick off disable sequence, unless climater keep-alive is holding
		// charging at min power — otherwise the "pausing soon" badge would
		// flash on/off forever while climater is active (issue #29834).
		if projectedSitePower >= lp.Disable.Threshold && !lp.vehicleClimateActive() {
			lp.log.DEBUG.Printf("projected site power %.0fW >= %.0fW disable threshold", projectedSitePower, lp.Disable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("pv disable timer start: %v", lp.GetDisableDelay())
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.GetDisableDelay(), pvDisable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.GetDisableDelay() {
				lp.log.DEBUG.Println("pv disable timer elapsed")

				// reset timer to prevent immediate charger re-enabling
				lp.resetPVTimer()

				return 0
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv disable timer remaining: %v", (lp.GetDisableDelay() - elapsed).Round(time.Second))
			}
		} else {
			// reset timer
			lp.resetPVTimer("disable")
		}

		// lp.log.DEBUG.Println("pv disable timer: keep enabled")
		return reachableMinPower
	}

	if mode == api.ModePV && !lp.enabled {
		// kick off enable sequence
		if (lp.Enable.Threshold == 0 && targetPower >= reachableMinPower) ||
			(lp.Enable.Threshold != 0 && sitePower <= lp.Enable.Threshold) {
			lp.log.DEBUG.Printf("site power %.0fW <= %.0fW enable threshold", sitePower, lp.Enable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("pv enable timer start: %v", lp.GetEnableDelay())
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.GetEnableDelay(), pvEnable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.GetEnableDelay() {
				lp.log.DEBUG.Println("pv enable timer elapsed")

				// reset timer to prevent immediate charger re-disabling
				lp.resetPVTimer()

				return reachableMinPower
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv enable timer remaining: %v", (lp.GetEnableDelay() - elapsed).Round(time.Second))
			}
		} else {
			// reset timer
			lp.resetPVTimer("enable")
		}

		// lp.log.DEBUG.Println("pv enable timer: keep disabled")
		return 0
	}

	// reset timer to disabled state
	lp.resetPVTimer()

	// cap at maximum power
	return min(targetPower, maxPower)
}
