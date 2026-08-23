package core

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
)

// enforcePhases scales to the configured phases
func (c *CurrentController) enforcePhases() error {
	return c.scalePhases(c.phasesConfigured)
}

// effectiveCurrent returns the currently effective charging current
func (c *CurrentController) effectiveCurrent() float64 {
	if !c.lp.charging() {
		return 0
	}

	// adjust actual current for vehicles like Zoe where it remains below target
	if c.chargeCurrents != nil {
		cur := max(c.chargeCurrents[0], c.chargeCurrents[1], c.chargeCurrents[2])
		return min(cur+2.0, c.offeredCurrent)
	}

	return c.offeredCurrent
}

// scalePhasesRequired validates if fixed phase configuration matches enabled phases
func (c *CurrentController) scalePhasesRequired() bool {
	return c.hasPhaseSwitching() && c.phasesConfigured != 0 && c.phasesConfigured != c.lp.GetPhases()
}

// scalePhasesIfAvailable scales if api.PhaseSwitcher is available and allowed
func (c *CurrentController) scalePhasesIfAvailable(phases int) error {
	if c.phasesConfigured != 0 {
		phases = c.phasesConfigured
	}

	if c.hasPhaseSwitching() {
		return c.scalePhases(phases)
	}

	return nil
}

// scalePhases adjusts the number of active phases and returns the appropriate charging current.
// Returns api.ErrNotAvailable if api.PhaseSwitcher is not available.
func (c *CurrentController) scalePhases(phases int) error {
	cp, ok := api.Cap[api.PhaseSwitcher](c.lp.charger)
	if !ok {
		panic("charger does not implement api.PhaseSwitcher")
	}

	if c.lp.GetPhases() != phases {
		// switch phases
		if err := cp.Phases1p3p(phases); err != nil {
			return fmt.Errorf("switch phases: %w", err)
		}

		c.lp.log.DEBUG.Printf("switched phases: %dp", phases)

		// prevent premature measurement of active phases
		c.phasesSwitched = c.lp.clock.Now()

		// update setting and reset timer
		c.SetPhases(phases)

		// some vehicles may hang on phase switch
		c.lp.startWakeUpTimer()
	}

	return nil
}

// circuitAllowsPhases checks if the circuit power limit allows charging at minCurrent on phases
func (c *CurrentController) circuitAllowsPhases(phases int, minCurrent float64) bool {
	if c.lp.circuit == nil {
		return true
	}

	minPower := currentToPower(minCurrent, phases)
	powerLimit := c.lp.circuit.ValidatePower(c.lp.chargePower, minPower)
	if powerLimit < minPower {
		c.lp.log.DEBUG.Printf("available circuit power %.0fW < %.0fW min %dp power", powerLimit, minPower, phases)
		return false
	}

	return true
}

// fastCharging scales to 3p if available and sets maximum current
func (c *CurrentController) fastCharging() error {
	if c.hasPhaseSwitching() {
		phases := 3

		// load management limit active
		if !c.circuitAllowsPhases(3, c.effectiveMinCurrent()) {
			phases = 1
		}

		// ignore api.ErrNotAvailable: the phase switch could not be performed
		// right now, continue with the current phase configuration
		if err := c.scalePhasesIfAvailable(phases); err != nil && !errors.Is(err, api.ErrNotAvailable) {
			return err
		}
	}

	return c.setLimit(c.effectiveMaxCurrent())
}

// minCharging scales to 1p if available and sets minimum current
func (c *CurrentController) minCharging() error {
	if c.hasPhaseSwitching() {
		// ignore api.ErrNotAvailable: the phase switch could not be performed
		// right now, continue with the current phase configuration
		if err := c.scalePhasesIfAvailable(1); err != nil && !errors.Is(err, api.ErrNotAvailable) {
			return err
		}
	}

	return c.setLimit(c.effectiveMinCurrent())
}

// pvScalePhases switches phases if necessary and returns number of phases switched to
func (c *CurrentController) pvScalePhases(sitePower, minCurrent, maxCurrent float64) int {
	phases := c.lp.GetPhases()

	// observed phase state inconsistency
	// - https://github.com/evcc-io/evcc/issues/1572
	// - https://github.com/evcc-io/evcc/issues/2230
	// - https://github.com/evcc-io/evcc/issues/2613
	measuredPhases := c.GetMeasuredPhases()
	if phases > 0 && phases < measuredPhases {
		if c.lp.chargerUpdateCompleted() && c.phaseSwitchCompleted() {
			c.lp.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", phases, measuredPhases)
		}
		c.ResetMeasuredPhases()
	}

	var waiting bool
	activePhases := c.ActivePhases()
	availablePower := c.lp.chargePower - sitePower
	scalable := activePhases > 1 && c.phasesConfigured < 3

	if scalable {
		insufficient := (sitePower > 0 || !c.enabled) && powerToCurrent(availablePower, activePhases) < minCurrent
		if insufficient {
			c.lp.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", availablePower, float64(activePhases)*Voltage*minCurrent, activePhases)
		}

		// while charging, scaling down only helps if 1p is sustainable, otherwise it
		// merely delays the pv disable timer by the phase timer duration
		useful := !c.enabled || !c.lp.charging() || powerToCurrent(availablePower, 1) >= minCurrent
		if insufficient && !useful {
			c.lp.log.DEBUG.Printf("available power %.0fW < %.0fW min 1p threshold, disabling instead of scaling down", availablePower, Voltage*minCurrent)
		}

		// scaling down also frees load management headroom for min power on activePhases
		scalable = insufficient && useful || !c.circuitAllowsPhases(activePhases, minCurrent)
	}

	// scale down phases
	if scalable {
		if !c.lp.charging() { // scale immediately if not charging
			c.phaseTimer = elapsed
		}

		if c.phaseTimer.IsZero() {
			c.lp.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			c.phaseTimer = c.lp.clock.Now()
		}

		c.lp.publishTimer(phaseTimer, c.lp.GetDisableDelay(), phaseScale1p)

		if elapsed := c.lp.clock.Since(c.phaseTimer); elapsed >= c.lp.GetDisableDelay() {
			if err := c.scalePhases(1); err != nil {
				// a charger may report it cannot switch phases right now
				// (api.ErrNotAvailable); assume a failed switch and stay silent
				if !errors.Is(err, api.ErrNotAvailable) {
					c.lp.log.ERROR.Println(err)
				}
				// switch did not complete - phase count is unchanged
				return phases
			}
			return 1
		}

		waiting = true
	}

	// load management may cap the 1p current far below the theoretical maximum
	if c.lp.circuit != nil {
		maxCurrent = c.lp.circuit.ValidateCurrent(c.actualMaxChargeCurrent(), maxCurrent)
	}

	maxPhases := c.MaxActivePhases()
	target1pCurrent := powerToCurrent(availablePower, 1)

	// scaling up is pointless unless load management allows min current and power on maxPhases
	scalable = maxPhases > 1 && phases < maxPhases && target1pCurrent > maxCurrent &&
		maxCurrent >= minCurrent && c.circuitAllowsPhases(maxPhases, minCurrent)

	// scale up phases
	if targetCurrent := powerToCurrent(availablePower, maxPhases); targetCurrent >= minCurrent && scalable {
		c.lp.log.DEBUG.Printf("available power %.0fW > %.0fW min %dp threshold", availablePower, float64(maxPhases)*Voltage*minCurrent, maxPhases)

		if !c.lp.charging() { // scale immediately if not charging
			c.phaseTimer = elapsed
		}

		if c.phaseTimer.IsZero() {
			c.lp.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			c.phaseTimer = c.lp.clock.Now()
		}

		c.lp.publishTimer(phaseTimer, c.lp.GetEnableDelay(), phaseScale3p)

		if elapsed := c.lp.clock.Since(c.phaseTimer); elapsed >= c.lp.GetEnableDelay() {
			if err := c.scalePhases(3); err != nil {
				// a charger may report it cannot switch phases right now
				// (api.ErrNotAvailable); assume a failed switch and stay silent
				if !errors.Is(err, api.ErrNotAvailable) {
					c.lp.log.ERROR.Println(err)
				}
				// switch did not complete - phase count is unchanged
				return phases
			}
			return 3
		}

		waiting = true
	}

	// reset timer to disabled state
	if !waiting && !c.phaseTimer.IsZero() {
		c.resetPhaseTimer()
	}

	return 0
}
