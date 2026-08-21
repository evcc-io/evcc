package core

import (
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
)

// setSuggestion stores the optimizer suggestion for the current slot
func (lp *Loadpoint) setSuggestion(s *types.Suggestion) {
	lp.Lock()
	defer lp.Unlock()

	lp.suggestion = s
	lp.suggestionUpdated = lp.clock.Now()
}

// optimizerControlled indicates that the optimizer decides for this loadpoint.
// Heating devices and switch sockets cannot be modelled and keep their limits,
// as do vehicles without known capacity- optimizerRequest skips them, too.
func (lp *Loadpoint) optimizerControlled() bool {
	if lp.site == nil || !lp.site.Automatic() ||
		lp.chargerHasFeature(api.Heating) || lp.chargerHasFeature(api.IntegratedDevice) {
		return false
	}

	v := lp.GetVehicle()
	return v != nil && v.Capacity() > 0
}

// gate returns the optimizer's start/stop decision for the current slot,
// nil if the optimizer does not control this loadpoint
func (lp *Loadpoint) gate() *types.Suggestion {
	if !lp.optimizerControlled() {
		return nil
	}

	lp.RLock()
	defer lp.RUnlock()

	// a stalled optimizer must not keep the loadpoint gated
	if lp.suggestion == nil || lp.clock.Since(lp.suggestionUpdated) > suggestionMaxAge {
		return nil
	}

	return lp.suggestion
}

// planDeadlineCritical returns true if the plan goal can only be reached by
// charging now. Backstops a plan the optimizer can no longer satisfy.
func (lp *Loadpoint) planDeadlineCritical() bool {
	planTime := lp.EffectivePlanTime()
	if planTime.IsZero() {
		return false
	}

	goal, _ := lp.GetPlanGoal()
	if goal <= 0 {
		return false
	}

	required := lp.GetPlanRequiredDuration(goal, lp.EffectiveMaxPower())

	// past the plan time the goal was missed- keep charging like plannerActive does
	if remaining := lp.clock.Until(planTime); remaining > 0 {
		return required >= remaining
	}

	return required > 0
}

// optimizerSurplus reports that the optimizer matched the charge power to the
// forecast surplus instead of feeding the loadpoint from the grid. Full power
// is grid-fed by definition and never counts as surplus.
func (lp *Loadpoint) optimizerSurplus(s *types.Suggestion) bool {
	return s.Charge < lp.EffectiveMaxPower()-suggestionThreshold &&
		math.Abs(s.Grid) <= suggestionThreshold
}

// optimizerCharging applies the optimizer's charging decision. It returns false
// if the loadpoint is to follow pv surplus instead, leaving current and phases
// to the regular pv control loop. Otherwise current and phases follow the
// suggested power, only the level is the optimizer's decision.
func (lp *Loadpoint) optimizerCharging(s *types.Suggestion, mode api.ChargeMode, welcomeCharge bool) (bool, error) {
	if s.Action == actionCharge && lp.optimizerSurplus(s) {
		// the forecast surplus only holds on average, so the pv loop tracks the
		// measured one- its enable/disable and phase timers must keep running
		if lp.pvTimer.Equal(elapsed) {
			// elapsed by a previous grid-fed slot, would disable on the first dip
			lp.resetPVTimer()
		}

		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), following pv surplus", s.Charge)
		return false, nil
	}

	lp.resetPhaseTimer()
	lp.elapsePVTimer() // let PV mode disable immediately afterwards

	if s.Action == actionCharge {
		// grid-fed charging at full power
		if s.Charge >= lp.EffectiveMaxPower()-suggestionThreshold {
			lp.log.DEBUG.Printf("optimizer: charge (%.0fW), full power", s.Charge)
			return true, lp.fastCharging()
		}

		// a limited setpoint, e.g. a minimum demand or a grid import limit
		if current := powerToCurrent(s.Charge, lp.ActivePhases()); current >= lp.effectiveMinCurrent() {
			lp.log.DEBUG.Printf("optimizer: charge (%.0fW)", s.Charge)
			return true, lp.setLimit(current)
		}

		// below what the active phases can deliver, minCharging scales down instead
		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), minimum power", s.Charge)
		return true, lp.minCharging()
	}

	// minpv keeps its minimum power, the optimizer plans with it
	if mode == api.ModeMinPV {
		lp.log.DEBUG.Println("optimizer: stop, keeping minimum power")
		return true, lp.minCharging()
	}

	// without welcome charge the vehicle never reports identity and soc, leaving
	// it unmodelled- the optimizer would then keep asking to stop
	if welcomeCharge {
		lp.log.DEBUG.Println("optimizer: stop, welcome charge")
		lp.resetPVTimer()
		return true, lp.setLimit(lp.effectiveMinCurrent())
	}

	if lp.vehicleClimateActive() {
		lp.log.DEBUG.Println("optimizer: stop, climate active")
		return true, lp.setLimit(lp.effectiveMinCurrent())
	}

	lp.log.DEBUG.Println("optimizer: stop")
	return true, lp.setLimit(0)
}
