package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/types"
)

// setSuggestion stores the optimizer suggestion for the current slot
func (lp *Loadpoint) setSuggestion(s *types.Suggestion) {
	lp.Lock()
	defer lp.Unlock()

	lp.suggestion = s
	lp.suggestionUpdated = lp.clock.Now()
}

// setOptimizerPlan stores the charging schedule of the last solve
func (lp *Loadpoint) setOptimizerPlan(plan optimizerPlan) {
	lp.Lock()
	defer lp.Unlock()

	lp.optimizerPlan = plan
}

// OptimizerPlan returns the optimizer's remaining charging schedule up to the
// plan time and its average power, nil if the optimizer is not in charge. Later
// slots serve surplus, not the plan.
func (lp *Loadpoint) OptimizerPlan(planTime time.Time) (api.Rates, float64) {
	if lp.gate() == nil {
		return nil, 0
	}

	lp.RLock()
	defer lp.RUnlock()

	now := lp.clock.Now()

	var rates api.Rates
	var energy float64
	for i, slot := range lp.optimizerPlan.rates {
		// keep what is left until the plan time, like the planner does: the schedule
		// is from the last solve and its goal slot may end after the plan time
		clipped := slot
		if clipped.Start.Before(now) {
			clipped.Start = now
		}
		if clipped.End.After(planTime) {
			clipped.End = planTime
		}
		if !clipped.Start.Before(clipped.End) {
			continue
		}

		rates = append(rates, clipped)
		energy += lp.optimizerPlan.energy[i] * float64(clipped.End.Sub(clipped.Start)) / float64(slot.End.Sub(slot.Start))
	}

	if len(rates) == 0 {
		return nil, 0
	}

	return rates, energy / planner.Duration(rates).Hours()
}

// optimizerControlled indicates that the optimizer decides for this loadpoint.
// Heating devices and switch sockets cannot be modelled and keep their limits,
// as do vehicles without known capacity- optimizerRequest skips them, too.
func (lp *Loadpoint) optimizerControlled() bool {
	if lp.site == nil || !lp.site.AutomaticLoadpoints() ||
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

// surplusRegime indicates that the suggested charge power matches the forecast
// surplus, leaving the loadpoint to the pv loop instead of pinning it to a setpoint.
// Full power is grid-fed by definition and never counts as surplus.
func (lp *Loadpoint) surplusRegime(s *types.Suggestion) bool {
	full := s.Charge >= lp.EffectiveMaxPower()-suggestionThreshold
	return s.Action == actionCharge && !full && math.Abs(s.Grid) <= suggestionThreshold
}

// optimizerCharging applies the optimizer's charging decision. It returns false
// if the loadpoint is to follow pv surplus instead, leaving current and phases
// to the regular pv control loop. Otherwise current and phases follow the
// suggested power, only the level is the optimizer's decision.
func (lp *Loadpoint) optimizerCharging(s *types.Suggestion) (bool, error) {
	if lp.surplusRegime(s) {
		// the charge power matches the forecast surplus, which only holds on average-
		// the pv loop tracks the measured one, so its timers must keep running
		if lp.pvTimer.Equal(elapsed) {
			// elapsed by a previous grid-fed slot, would disable on the first dip
			lp.resetPVTimer()
		}

		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), following pv surplus", s.Charge)
		return false, nil
	}

	if s.Action != actionCharge && !lp.GetAlwaysCharge().Active() {
		// a stop rests on the forecast surplus, which may be too low- the pv loop still
		// enables on measured surplus. A timer elapsed by a grid-fed slot stops at once.
		if !lp.enabled && lp.pvTimer.Equal(elapsed) {
			lp.resetPVTimer()
		}

		lp.log.DEBUG.Println("optimizer: stop, following measured pv surplus")
		return false, nil
	}

	lp.elapsePVTimer() // let PV mode disable immediately afterwards

	// full power, or more than the active phases can deliver: fastCharging scales
	// up and owns the phase timer, resetting it here would restart the delay every cycle
	full := s.Charge >= lp.EffectiveMaxPower()-suggestionThreshold
	if s.Action == actionCharge && (full || s.Charge > currentToPower(lp.effectiveMaxCurrent(), lp.ActivePhases())) {
		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), full power", s.Charge)
		return true, lp.fastCharging()
	}

	lp.resetPhaseTimer()

	if s.Action == actionCharge {
		// a limited setpoint, e.g. a minimum demand or a grid import limit
		if current := powerToCurrent(s.Charge, lp.ActivePhases()); current >= lp.effectiveMinCurrent() {
			lp.log.DEBUG.Printf("optimizer: charge (%.0fW)", s.Charge)
			return true, lp.setLimit(current)
		}

		// below what the active phases can deliver, minCharging scales down instead
		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), minimum power", s.Charge)
		return true, lp.minCharging()
	}

	// always charge keeps its minimum power, the optimizer plans with it
	lp.log.DEBUG.Println("optimizer: stop, keeping minimum power")
	return true, lp.minCharging()
}
