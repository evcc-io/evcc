package core

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
)

// enforcePhases scales to the configured phases
func (lp *CurrentController) enforcePhases() error {
	return lp.scalePhases(lp.phasesConfigured)
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
