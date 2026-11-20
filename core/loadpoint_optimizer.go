package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/grid"
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
	if lp.activeSuggestion() == nil {
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

// optimizerPlanGap keeps a running plan charging through a gap of up to one slot before the
// optimizer's next plan slot, like the planner avoids a re-start. At equal prices the optimizer
// may leave any single slot empty, which costs a stop and start of the vehicle. A full slot is
// only bridged if it is no more expensive than the next plan slot, else it was skipped for its price.
func (lp *Loadpoint) optimizerPlanGap() bool {
	if !lp.planActive {
		return false
	}

	plan, _ := lp.OptimizerPlan(lp.EffectivePlanTime())
	if plan == nil || !planner.SlotAt(lp.clock.Now(), plan).End.IsZero() {
		return false
	}

	start := planner.Start(plan)
	gap := lp.clock.Until(start)
	if gap < tariff.SlotDuration-time.Minute {
		return true
	}
	if gap > tariff.SlotDuration {
		return false
	}

	rates := tariff.Rates(lp.site.GetTariff(api.TariffUsagePlanner))
	current, err := rates.At(lp.clock.Now())
	if err != nil {
		return false
	}

	next, err := rates.At(start)
	return err == nil && current.Value <= next.Value
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

// fixedDemandModelled indicates that optimizerRequest contains the charging as fixed vehicle
// demand (mode now, smart cost limit, planner schedule) even if the optimizer does not control
// the loadpoint. Min soc charging follows evcc's own timing, which the request does not model.
func (lp *Loadpoint) fixedDemandModelled() bool {
	maxPower := lp.EffectiveMaxPower()
	if lp.chargerHasFeature(api.Heating) || lp.chargerHasFeature(api.IntegratedDevice) || maxPower <= 0 {
		return false
	}

	if v := lp.GetVehicle(); v == nil || (v.Capacity() == 0 && lp.GetLimitEnergy() == 0) {
		return false
	}

	// a 100% plan charging on past its goal has no planner schedule
	goal, socBased := lp.GetPlanGoal()
	continues := socBased && goal == 100 && lp.GetPlanRequiredDuration(goal, maxPower) <= 0

	lp.RLock()
	defer lp.RUnlock()

	return !(lp.planActive && continues) && !lp.minSocNotReached()
}

// activeSuggestion returns the optimizer's start/stop decision for the current slot,
// nil if the optimizer does not control this loadpoint
func (lp *Loadpoint) activeSuggestion() *types.Suggestion {
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

	goal, isSocBased := lp.GetPlanGoal()
	if goal <= 0 {
		return false
	}

	required := lp.GetPlanRequiredDuration(goal, lp.EffectiveMaxPower())
	if required <= 0 && lp.fullSocPlanContinues(goal, isSocBased) {
		return true
	}

	// past the plan time the goal was missed- keep charging like plannerActive does
	critical := required > 0
	if remaining := lp.clock.Until(planTime); remaining > 0 {
		critical = required >= remaining
	}

	// hold a critical verdict for one slot, like the planner holds its slot: while charging
	// the soc estimate may shrink the required duration faster than the clock (taper model)
	if critical {
		lp.criticalSlotEnd = lp.clock.Now().Add(tariff.SlotDuration)
	}

	return critical || lp.clock.Now().Before(lp.criticalSlotEnd)
}

// surplusRegime indicates that the suggested charge power matches the forecast
// surplus, leaving the loadpoint to the pv loop instead of pinning it to a setpoint.
// Full power is grid-fed by definition and never counts as surplus, nor does a
// charge the home battery covers: the pv loop sees its discharge as deficit.
func (lp *Loadpoint) surplusRegime(s *types.Suggestion) bool {
	full := s.Charge >= lp.EffectiveMaxPower()-suggestionThreshold
	return s.Action == actionCharge && !full && math.Abs(s.Grid) <= suggestionThreshold && s.BatteryDischarge <= suggestionThreshold
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
	// up and owns the phase timer, resetting it here would restart the delay every cycle.
	// Inside the 1p/3p gap neither phase count matches the suggestion: stay below it on 1p
	// instead of scaling up to a 3p minimum that draws unplanned power, e.g. from grid.
	full := s.Charge >= lp.EffectiveMaxPower()-suggestionThreshold
	scaleUp := s.Charge > grid.CurrentToPower(lp.effectiveMaxCurrent(), lp.ActivePhases()) &&
		s.Charge >= grid.CurrentToPower(lp.effectiveMinCurrent(), 3)
	if s.Action == actionCharge && (full || scaleUp) {
		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), max current", s.Charge)
		return true, lp.fastCharging()
	}

	lp.resetPhaseTimer()

	if s.Action == actionCharge {
		// a limited setpoint, e.g. a minimum demand or a grid import limit
		if current := grid.PowerToCurrent(s.Charge, lp.ActivePhases()); current >= lp.effectiveMinCurrent() {
			lp.log.DEBUG.Printf("optimizer: charge (%.0fW)", s.Charge)
			return true, lp.setLimit(min(current, lp.effectiveMaxCurrent()))
		}

		// below what the active phases can deliver, minCharging scales down instead
		lp.log.DEBUG.Printf("optimizer: charge (%.0fW), minimum power", s.Charge)
		return true, lp.minCharging()
	}

	// always charge keeps its minimum power, the optimizer plans with it
	lp.log.DEBUG.Println("optimizer: stop, keeping minimum power")
	return true, lp.minCharging()
}
