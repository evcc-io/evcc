package core

import (
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	pvTimer   = "pv"
	pvEnable  = "enable"
	pvDisable = "disable"
)

// pvMaxCurrent calculates the maximum target current for PV mode
func (lp *Loadpoint) pvMaxCurrent(mode api.ChargeMode, sitePower float64, batteryBuffered, batteryStart bool) float64 {
	// read only once to simplify testing
	minCurrent := lp.GetMinCurrent()
	maxCurrent := lp.GetMaxCurrent()

	// switch phases up/down
	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		availablePower := -sitePower + lp.chargePower
		_ = lp.pvScalePhases(availablePower, minCurrent, maxCurrent)
	}

	// calculate target charge current from delta power and actual current
	effectiveCurrent := lp.effectiveCurrent()
	activePhases := lp.activePhases()
	deltaCurrent := powerToCurrent(-sitePower, activePhases)
	targetCurrent := math.Max(effectiveCurrent+deltaCurrent, 0)

	lp.log.DEBUG.Printf("pv charge current: %.3gA = %.3gA + %.3gA (%.0fW @ %dp)", targetCurrent, effectiveCurrent, deltaCurrent, sitePower, activePhases)

	// in MinPV mode or under special conditions return at least minCurrent
	if (mode == api.ModeMinPV || batteryStart || batteryBuffered && lp.charging()) && targetCurrent < minCurrent {
		return minCurrent
	}

	if mode == api.ModePV && lp.enabled && targetCurrent < minCurrent {
		// kick off disable sequence
		if sitePower >= lp.Disable.Threshold && lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("site power %.0fW >= %.0fW disable threshold", sitePower, lp.Disable.Threshold)

			if lp.pvTimer.IsZero() {
				lp.log.DEBUG.Printf("pv disable timer start: %v", lp.Disable.Delay)
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.Disable.Delay, pvDisable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.Disable.Delay {
				lp.log.DEBUG.Println("pv disable timer elapsed")
				return 0
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv disable timer remaining: %v", (lp.Disable.Delay - elapsed).Round(time.Second))
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
				lp.log.DEBUG.Printf("pv enable timer start: %v", lp.Enable.Delay)
				lp.pvTimer = lp.clock.Now()
			}

			lp.publishTimer(pvTimer, lp.Enable.Delay, pvEnable)

			elapsed := lp.clock.Since(lp.pvTimer)
			if elapsed >= lp.Enable.Delay {
				lp.log.DEBUG.Println("pv enable timer elapsed")
				return minCurrent
			}

			// suppress duplicate log message after timer started
			if elapsed > time.Second {
				lp.log.DEBUG.Printf("pv enable timer remaining: %v", (lp.Enable.Delay - elapsed).Round(time.Second))
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
	targetCurrent = math.Min(targetCurrent, maxCurrent)

	return targetCurrent
}

// pvScalePhases switches phases if necessary and returns if switch occurred
func (lp *Loadpoint) pvScalePhases(availablePower, minCurrent, maxCurrent float64) bool {
	phases := lp.GetPhases()

	// observed phase state inconsistency
	// - https://github.com/evcc-io/evcc/issues/1572
	// - https://github.com/evcc-io/evcc/issues/2230
	// - https://github.com/evcc-io/evcc/issues/2613
	measuredPhases := lp.getMeasuredPhases()
	if phases > 0 && phases < measuredPhases {
		if lp.guardGracePeriodElapsed() {
			lp.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", phases, measuredPhases)
		}
		lp.resetMeasuredPhases()
	}

	var waiting bool
	activePhases := lp.activePhases()

	// scale down phases
	if targetCurrent := powerToCurrent(availablePower, activePhases); targetCurrent < minCurrent && activePhases > 1 && lp.ConfiguredPhases < 3 {
		lp.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", availablePower, float64(activePhases)*Voltage*minCurrent, activePhases)

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.Disable.Delay, phaseScale1p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.Disable.Delay {
			lp.log.DEBUG.Printf("phase %s timer elapsed", phaseScale1p)
			if err := lp.scalePhases(1); err == nil {
				lp.log.DEBUG.Printf("switched phases: 1p @ %.0fW", availablePower)
			} else {
				lp.log.ERROR.Println(err)
			}
			return true
		}

		waiting = true
	}

	maxPhases := lp.maxActivePhases()
	target1pCurrent := powerToCurrent(availablePower, 1)
	scalable := maxPhases > 1 && phases < maxPhases && target1pCurrent > maxCurrent

	// scale up phases
	if targetCurrent := powerToCurrent(availablePower, maxPhases); targetCurrent >= minCurrent && scalable {
		lp.log.DEBUG.Printf("available power %.0fW > %.0fW min %dp threshold", availablePower, 3*Voltage*minCurrent, maxPhases)

		if lp.phaseTimer.IsZero() {
			lp.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			lp.phaseTimer = lp.clock.Now()
		}

		lp.publishTimer(phaseTimer, lp.Enable.Delay, phaseScale3p)

		if elapsed := lp.clock.Since(lp.phaseTimer); elapsed >= lp.Enable.Delay {
			lp.log.DEBUG.Printf("phase %s timer elapsed", phaseScale3p)
			if err := lp.scalePhases(3); err == nil {
				lp.log.DEBUG.Printf("switched phases: 3p @ %.0fW", availablePower)
			} else {
				lp.log.ERROR.Println(err)
			}
			return true
		}

		waiting = true
	}

	// reset timer to disabled state
	if !waiting && !lp.phaseTimer.IsZero() {
		lp.resetPhaseTimer()
	}

	return false
}

// elapsePVTimer puts the pv enable/disable timer into elapsed state
func (lp *Loadpoint) elapsePVTimer() {
	if lp.pvTimer.Equal(elapsed) {
		return
	}

	lp.log.DEBUG.Printf("pv timer elapse")

	lp.pvTimer = elapsed
	lp.publishTimer(pvTimer, 0, timerInactive)

	lp.elapseGuard()
}

// resetPVTimer resets the pv enable/disable timer to disabled state
func (lp *Loadpoint) resetPVTimer(typ ...string) {
	if lp.pvTimer.IsZero() {
		return
	}

	msg := "pv timer reset"
	if len(typ) == 1 {
		msg = fmt.Sprintf("pv %s timer reset", typ[0])
	}
	lp.log.DEBUG.Printf(msg)

	lp.pvTimer = time.Time{}
	lp.publishTimer(pvTimer, 0, timerInactive)
}
