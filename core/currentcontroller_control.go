package core

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
)

// chargeMinimum charges at the effective minimum current
func (lp *CurrentController) chargeMinimum() error {
	return lp.setLimit(lp.effectiveMinCurrent())
}

// enforcePhases scales to the configured phases
func (lp *CurrentController) enforcePhases() error {
	return lp.scalePhases(lp.phasesConfigured)
}

// pvMaxPower returns the PV mode target power (0 disables)
func (lp *CurrentController) pvMaxPower(mode api.ChargeMode, sitePower, batteryBoostPower float64, batteryBuffered, batteryStart bool) float64 {
	return currentToPower(lp.pvMaxCurrent(mode, sitePower, batteryBoostPower, batteryBuffered, batteryStart), lp.ActivePhases())
}

// effectiveCurrent returns the currently effective charging current
func (lp *CurrentController) effectiveCurrent() float64 {
	if !lp.charging() {
		return 0
	}

	// adjust actual current for vehicles like Zoe where it remains below target
	if lp.chargeCurrents != nil {
		cur := max(lp.chargeCurrents[0], lp.chargeCurrents[1], lp.chargeCurrents[2])
		return min(cur+2.0, lp.offeredCurrent)
	}

	return lp.offeredCurrent
}

// scalePhasesRequired validates if fixed phase configuration matches enabled phases
func (lp *CurrentController) scalePhasesRequired() bool {
	return lp.hasPhaseSwitching() && lp.phasesConfigured != 0 && lp.phasesConfigured != lp.GetPhases()
}

// scalePhasesIfAvailable scales if api.PhaseSwitcher is available and allowed
func (lp *CurrentController) scalePhasesIfAvailable(phases int) error {
	if lp.phasesConfigured != 0 {
		phases = lp.phasesConfigured
	}

	if lp.hasPhaseSwitching() {
		return lp.scalePhases(phases)
	}

	return nil
}

// scalePhases adjusts the number of active phases and returns the appropriate charging current.
// Returns api.ErrNotAvailable if api.PhaseSwitcher is not available.
func (lp *CurrentController) scalePhases(phases int) error {
	cp, ok := api.Cap[api.PhaseSwitcher](lp.charger)
	if !ok {
		panic("charger does not implement api.PhaseSwitcher")
	}

	if lp.GetPhases() != phases {
		// switch phases
		if err := cp.Phases1p3p(phases); err != nil {
			return fmt.Errorf("switch phases: %w", err)
		}

		lp.log.DEBUG.Printf("switched phases: %dp", phases)

		// prevent premature measurement of active phases
		lp.phasesSwitched = lp.clock.Now()

		// update setting and reset timer
		lp.SetPhases(phases)

		// some vehicles may hang on phase switch
		lp.startWakeUpTimer()
	}

	return nil
}

// fastCharging scales to 3p if available and sets maximum current
func (lp *CurrentController) fastCharging() error {
	if lp.hasPhaseSwitching() {
		phases := 3

		// load management limit active
		if lp.circuit != nil {
			minPower3p := currentToPower(lp.effectiveMinCurrent(), 3)
			if powerLimit := lp.circuit.ValidatePower(lp.chargePower, minPower3p); powerLimit < minPower3p {
				phases = 1
				lp.log.DEBUG.Printf("fast charging: scaled to 1p to match %.0fW available circuit power", powerLimit)
			}
		}

		// ignore api.ErrNotAvailable: the phase switch could not be performed
		// right now, continue with the current phase configuration
		if err := lp.scalePhasesIfAvailable(phases); err != nil && !errors.Is(err, api.ErrNotAvailable) {
			return err
		}
	}

	return lp.setLimit(lp.effectiveMaxCurrent())
}

// minCharging scales to 1p if available and sets minimum current
func (lp *CurrentController) minCharging() error {
	if lp.hasPhaseSwitching() {
		// ignore api.ErrNotAvailable: the phase switch could not be performed
		// right now, continue with the current phase configuration
		if err := lp.scalePhasesIfAvailable(1); err != nil && !errors.Is(err, api.ErrNotAvailable) {
			return err
		}
	}

	return lp.setLimit(lp.effectiveMinCurrent())
}

// pvScalePhases switches phases if necessary and returns number of phases switched to
func (lp *CurrentController) pvScalePhases(sitePower, minCurrent, maxCurrent float64) int {
	phases := lp.GetPhases()

	// observed phase state inconsistency
	// - https://github.com/evcc-io/evcc/issues/1572
	// - https://github.com/evcc-io/evcc/issues/2230
	// - https://github.com/evcc-io/evcc/issues/2613
	measuredPhases := lp.GetMeasuredPhases()
	if phases > 0 && phases < measuredPhases {
		if lp.chargerUpdateCompleted() && lp.phaseSwitchCompleted() {
			lp.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", phases, measuredPhases)
		}
		lp.ResetMeasuredPhases()
	}

	var waiting bool
	activePhases := lp.ActivePhases()
	availablePower := lp.chargePower - sitePower
	scalable := (sitePower > 0 || !lp.enabled) && activePhases > 1 && lp.phasesConfigured < 3

	// scale down phases
	if targetCurrent := powerToCurrent(availablePower, activePhases); targetCurrent < minCurrent && scalable {
		lp.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", availablePower, float64(activePhases)*Voltage*minCurrent, activePhases)

		if !lp.charging() { // scale immediately if not charging
			lp.phaseTimer = elapsed
		}

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.GetDisableDelay(), phaseScale1p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.GetDisableDelay() {
			if err := lp.scalePhases(1); err != nil {
				// a charger may report it cannot switch phases right now
				// (api.ErrNotAvailable); assume a failed switch and stay silent
				if !errors.Is(err, api.ErrNotAvailable) {
					lp.log.ERROR.Println(err)
				}
				// switch did not complete - phase count is unchanged
				return phases
			}
			return 1
		}

		waiting = true
	}

	maxPhases := lp.MaxActivePhases()
	target1pCurrent := powerToCurrent(availablePower, 1)
	scalable = maxPhases > 1 && phases < maxPhases && target1pCurrent > maxCurrent

	// scale up phases
	if targetCurrent := powerToCurrent(availablePower, maxPhases); targetCurrent >= minCurrent && scalable {
		lp.log.DEBUG.Printf("available power %.0fW > %.0fW min %dp threshold", availablePower, float64(maxPhases)*Voltage*minCurrent, maxPhases)

		if !lp.charging() { // scale immediately if not charging
			lp.phaseTimer = elapsed
		}

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.GetEnableDelay(), phaseScale3p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.GetEnableDelay() {
			if err := lp.scalePhases(3); err != nil {
				// a charger may report it cannot switch phases right now
				// (api.ErrNotAvailable); assume a failed switch and stay silent
				if !errors.Is(err, api.ErrNotAvailable) {
					lp.log.ERROR.Println(err)
				}
				// switch did not complete - phase count is unchanged
				return phases
			}
			return 3
		}

		waiting = true
	}

	// reset timer to disabled state
	if !waiting && !lp.phaseTimer.IsZero() {
		lp.resetPhaseTimer()
	}

	return 0
}

// boostPower returns the additional power that the loadpoint should draw from the battery
func (lp *CurrentController) boostPower(batteryBoostPower float64) float64 {
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

// pvMaxCurrent calculates the maximum target current for PV mode
func (lp *CurrentController) pvMaxCurrent(mode api.ChargeMode, sitePower, batteryBoostPower float64, batteryBuffered, batteryStart bool) float64 {
	// read only once to simplify testing
	minCurrent := lp.effectiveMinCurrent()
	maxCurrent := lp.effectiveMaxCurrent()

	// push demand to drain battery
	sitePower -= lp.boostPower(batteryBoostPower)

	// switch phases up/down
	var scaledTo int
	if lp.hasPhaseSwitching() && lp.phaseSwitchCompleted() {
		scaledTo = lp.pvScalePhases(sitePower, minCurrent, maxCurrent)
	}

	// calculate target charge current from delta power and actual current
	activePhases := lp.ActivePhases()
	effectiveCurrent := lp.effectiveCurrent()
	if scaledTo == 3 {
		// if we did scale, adjust the effective current to the new phase count
		effectiveCurrent /= float64(lp.maxActivePhases())
	}
	if lp.chargerHasFeature(api.IntegratedDevice) {
		// for slow-acting heating devices, only take actually consumed power into account
		effectiveCurrent = powerToCurrent(lp.chargePower, activePhases)
	}
	deltaCurrent := powerToCurrent(-sitePower, activePhases)
	targetCurrent := max(effectiveCurrent+deltaCurrent, 0)

	// in MinPV mode or under special conditions return at least minCurrent
	if battery := batteryStart || batteryBuffered && lp.charging(); (mode == api.ModeMinPV || battery) && targetCurrent < minCurrent {
		lp.log.DEBUG.Printf("pv charge current: min %.3gA > %.3gA (%.0fW @ %dp, battery: %t)", minCurrent, targetCurrent, sitePower, activePhases, battery)
		return minCurrent
	}

	lp.log.DEBUG.Printf("pv charge current: %.3gA = %.3gA + %.3gA (%.0fW @ %dp)", targetCurrent, effectiveCurrent, deltaCurrent, sitePower, activePhases)

	if mode == api.ModePV && lp.enabled && targetCurrent < minCurrent {
		projectedSitePower := sitePower
		if lp.hasPhaseSwitching() && !lp.phaseTimer.IsZero() {
			// calculate site power after a phase switch from activePhases phases -> 1 phase
			// notes: activePhases can be 1, 2 or 3 and phaseTimer can only be active if lp current is already at minCurrent
			projectedSitePower -= Voltage * minCurrent * float64(activePhases-1)
		}
		// kick off disable sequence, unless climater keep-alive is holding
		// charging at minCurrent — otherwise the "pausing soon" badge would
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
		return minCurrent
	}

	if mode == api.ModePV && !lp.enabled {
		// kick off enable sequence
		if (lp.Enable.Threshold == 0 && targetCurrent >= minCurrent) ||
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

				return minCurrent
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

	// cap at maximum current
	targetCurrent = min(targetCurrent, maxCurrent)

	return targetCurrent
}
