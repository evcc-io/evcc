package core

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
)

// enforcePhases scales to the configured phases
func (c *CurrentController) enforcePhases() error {
	return c.scalePhases(c.lp.phasesConfigured)
}

// effectiveCurrent returns the currently effective charging current
func (c *CurrentController) effectiveCurrent() float64 {
	if !c.lp.charging() {
		return 0
	}

	// adjust actual current for vehicles like Zoe where it remains below target
	if c.lp.chargeCurrents != nil {
		cur := max(c.lp.chargeCurrents[0], c.lp.chargeCurrents[1], c.lp.chargeCurrents[2])
		return min(cur+2.0, c.lp.offeredCurrent)
	}

	return c.lp.offeredCurrent
}

// scalePhasesRequired validates if fixed phase configuration matches enabled phases
func (c *CurrentController) scalePhasesRequired() bool {
	return c.lp.hasPhaseSwitching() && c.lp.phasesConfigured != 0 && c.lp.phasesConfigured != c.lp.GetPhases()
}

// scalePhasesIfAvailable scales if api.PhaseSwitcher is available and allowed
func (c *CurrentController) scalePhasesIfAvailable(phases int) error {
	if c.lp.phasesConfigured != 0 {
		phases = c.lp.phasesConfigured
	}

	if c.lp.hasPhaseSwitching() {
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
		c.lp.phasesSwitched = c.lp.clock.Now()

		// update setting and reset timer
		c.lp.SetPhases(phases)

		// some vehicles may hang on phase switch
		c.lp.startWakeUpTimer()
	}

	return nil
}

// fastCharging scales to 3p if available and sets maximum current
func (c *CurrentController) fastCharging() error {
	if c.lp.hasPhaseSwitching() {
		phases := 3

		// load management limit active
		if c.lp.circuit != nil {
			minPower3p := currentToPower(c.effectiveMinCurrent(), 3)
			if powerLimit := c.lp.circuit.ValidatePower(c.lp.chargePower, minPower3p); powerLimit < minPower3p {
				phases = 1
				c.lp.log.DEBUG.Printf("fast charging: scaled to 1p to match %.0fW available circuit power", powerLimit)
			}
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
	if c.lp.hasPhaseSwitching() {
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
	measuredPhases := c.lp.GetMeasuredPhases()
	if phases > 0 && phases < measuredPhases {
		if c.lp.chargerUpdateCompleted() && c.lp.phaseSwitchCompleted() {
			c.lp.log.WARN.Printf("ignoring inconsistent phases: %dp < %dp observed active", phases, measuredPhases)
		}
		c.lp.ResetMeasuredPhases()
	}

	var waiting bool
	activePhases := c.lp.ActivePhases()
	availablePower := c.lp.chargePower - sitePower
	scalable := (sitePower > 0 || !c.lp.enabled) && activePhases > 1 && c.lp.phasesConfigured < 3

	// scale down phases
	if targetCurrent := powerToCurrent(availablePower, activePhases); targetCurrent < minCurrent && scalable {
		c.lp.log.DEBUG.Printf("available power %.0fW < %.0fW min %dp threshold", availablePower, float64(activePhases)*Voltage*minCurrent, activePhases)

		if !c.lp.charging() { // scale immediately if not charging
			c.lp.phaseTimer = elapsed
		}

		if c.lp.phaseTimer.IsZero() {
			c.lp.log.DEBUG.Printf("start phase %s timer", phaseScale1p)
			c.lp.phaseTimer = c.lp.clock.Now()
		}

		c.lp.publishTimer(phaseTimer, c.lp.GetDisableDelay(), phaseScale1p)

		if elapsed := c.lp.clock.Since(c.lp.phaseTimer); elapsed >= c.lp.GetDisableDelay() {
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

	maxPhases := c.lp.MaxActivePhases()
	target1pCurrent := powerToCurrent(availablePower, 1)
	scalable = maxPhases > 1 && phases < maxPhases && target1pCurrent > maxCurrent

	// scale up phases
	if targetCurrent := powerToCurrent(availablePower, maxPhases); targetCurrent >= minCurrent && scalable {
		c.lp.log.DEBUG.Printf("available power %.0fW > %.0fW min %dp threshold", availablePower, float64(maxPhases)*Voltage*minCurrent, maxPhases)

		if !c.lp.charging() { // scale immediately if not charging
			c.lp.phaseTimer = elapsed
		}

		if c.lp.phaseTimer.IsZero() {
			c.lp.log.DEBUG.Printf("start phase %s timer", phaseScale3p)
			c.lp.phaseTimer = c.lp.clock.Now()
		}

		c.lp.publishTimer(phaseTimer, c.lp.GetEnableDelay(), phaseScale3p)

		if elapsed := c.lp.clock.Since(c.lp.phaseTimer); elapsed >= c.lp.GetEnableDelay() {
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
	if !waiting && !c.lp.phaseTimer.IsZero() {
		c.lp.resetPhaseTimer()
	}

	return 0
}
