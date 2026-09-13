package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
)

// boostPower returns the additional power that the loadpoint should draw from the battery
func (lp *Loadpoint) boostPower(ctrl *CurrentController, env Envelope, batteryPower float64) float64 {
	boost := lp.GetBatteryBoost()
	if boost == boostDisabled || boost == boostHold {
		return 0
	}

	// push demand to drain battery (at least 100W)
	delta := math.Max(100, math.Abs(lp.site.GetResidualPower()))

	if env.Coarse {
		// add step power to delta to make sure to step up to the next full amp
		// just using the step power as delta is not enough because this will result
		// in a too low current when there is a bit remaining grid consumption due to the accuracy
		// of the battery controller
		delta += env.Step
	}

	// bridge the power gap between 1p max and 3p min so pvScalePhases can trigger a scale-up
	if lp.site.GetBatteryMaxDischargePower() != nil {
		delta += ctrl.phaseSwitchGapPower()
	}

	// start boosting by setting maximum power
	if boost == boostStart {
		delta = lp.EffectiveMaxPower()

		// expire timers
		ctrl.expirePhaseTimer()
		lp.pvTimer = elapsed

		if lp.charging() {
			lp.setBatteryBoost(boostContinue)
		}
	}

	if maxDischargePower := lp.site.GetBatteryMaxDischargePower(); maxDischargePower != nil {
		// limit delta to what the battery can still provide
		delta = min(delta, max(0, *maxDischargePower-batteryPower))
	}

	res := max(0, batteryPower) + delta + lp.site.GetResidualPower()
	lp.log.DEBUG.Printf("pv charge battery boost: %.0fW = -%.0fW battery - %.0fW boost - %.0fW residual", -res, max(0, batteryPower), delta, lp.site.GetResidualPower())

	return res
}

// customThresholds indicates manually configured enable/disable thresholds that take precedence over the solar share
func (lp *Loadpoint) customThresholds() bool {
	return lp.Enable.Threshold != 0 || lp.Disable.Threshold != 0
}

// pvDisableThreshold returns the pv disable switch point, derived from the solar share unless
// custom thresholds are configured. minPower is the min power charging is expected to continue at.
func (lp *Loadpoint) pvDisableThreshold(minPower float64) float64 {
	if lp.customThresholds() {
		return lp.Disable.Threshold
	}
	// allow grid import for the non-solar part of the min power
	return (1 - lp.GetSolarShare()) * minPower
}

// pvEnableDecision returns the pv enable switch point and whether charging should start, derived
// from the solar share unless custom thresholds are configured. availablePower is unclamped.
func (lp *Loadpoint) pvEnableDecision(availablePower, minPower, sitePower float64) (float64, bool) {
	if !lp.customThresholds() {
		// require the solar share of the min power to come from surplus
		share := lp.GetSolarShare()
		return -share * minPower, availablePower >= share*minPower
	}

	threshold := lp.Enable.Threshold
	return threshold, (threshold == 0 && availablePower >= minPower) ||
		(threshold != 0 && sitePower <= threshold)
}

// pvTargetPower calculates the target charging power for smart mode (0 disables)
func (lp *Loadpoint) pvTargetPower(ctrl *CurrentController, sitePower, batteryPower float64, batteryBuffered, batteryStart bool) float64 {
	// snapshot the controller's capabilities once per cycle
	env := ctrl.Envelope()
	minPower := env.ActiveMin
	maxPower := env.ActiveMax
	reachableMinPower := env.ReachableMin
	alwaysCharge := lp.GetAlwaysCharge().Active()

	// push demand to drain battery
	sitePower -= lp.boostPower(ctrl, env, batteryPower)

	// always charge and the battery conditions hold charging at min power, no disable can follow
	battery := batteryStart || batteryBuffered && lp.charging() || lp.GetBatteryBoost() == boostContinue
	mayDisable := !alwaysCharge && !battery

	// provide surplus for phase reconciliation by the controller
	ctrl.Prepare(sitePower, mayDisable)

	// calculate target charge power from delta power and actual power
	availablePower := env.Effective - sitePower
	targetPower := max(availablePower, 0)

	// with always charge or under special conditions return at least min power
	if (alwaysCharge || battery) && targetPower < minPower {
		lp.log.DEBUG.Printf("pv charge power: min %.0fW > %.0fW (%.0fW @ %dp, battery: %t)", minPower, targetPower, sitePower, env.ActivePhases, battery)
		return reachableMinPower
	}

	lp.log.DEBUG.Printf("pv charge power: %.0fW = %.0fW - %.0fW (@ %dp)", targetPower, env.Effective, sitePower, env.ActivePhases)

	if !alwaysCharge && env.Enabled && targetPower < minPower {
		projectedSitePower := sitePower
		projectedMinPower := minPower
		// read live: boostPower may have expired the phase timer after the snapshot
		if ctrl.phaseScalePending() {
			// calculate site power after a phase switch to the minimum reachable phases
			// notes: phase timer can only be active if lp current is already at min current
			projectedSitePower -= minPower - reachableMinPower
			projectedMinPower = reachableMinPower
		}
		// a continuous device consuming less than its min power demand keeps the
		// remainder out of site power, hiding insufficient surplus until it ramps
		// up (#32282). Project the shortfall towards min power into the gate.
		if lp.chargerHasFeature(api.Continuous) {
			projectedSitePower += max(0, env.EffectiveMin-lp.chargePower)
		}

		disableThreshold := lp.pvDisableThreshold(projectedMinPower)

		// kick off disable sequence, unless climater keep-alive is holding
		// charging at min power — otherwise the "pausing soon" badge would
		// flash on/off forever while climater is active (issue #29834).
		if projectedSitePower >= disableThreshold && !lp.vehicleClimateActive() {
			lp.log.DEBUG.Printf("projected site power %.0fW >= %.0fW disable threshold", projectedSitePower, disableThreshold)

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

	if !alwaysCharge && !env.Enabled {
		enableThreshold, shouldEnable := lp.pvEnableDecision(availablePower, reachableMinPower, sitePower)

		// kick off enable sequence
		if shouldEnable {
			lp.log.DEBUG.Printf("site power %.0fW <= %.0fW enable threshold", sitePower, enableThreshold)

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
