package core

import (
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
// Heating devices and switch sockets cannot be modelled and keep their limits.
func (lp *Loadpoint) optimizerControlled() bool {
	return lp.site != nil && lp.site.Automatic() &&
		!lp.chargerHasFeature(api.Heating) && !lp.chargerHasFeature(api.IntegratedDevice)
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

	remaining := lp.clock.Until(planTime)

	return remaining > 0 && lp.GetPlanRequiredDuration(goal, lp.EffectiveMaxPower()) >= remaining
}

// optimizerCharging applies the optimizer's start/stop decision. Current and
// phases remain the loadpoint's decision.
func (lp *Loadpoint) optimizerCharging(s *types.Suggestion, mode api.ChargeMode) error {
	defer func() {
		lp.resetPhaseTimer()
		lp.elapsePVTimer() // let PV mode disable immediately afterwards
	}()

	if s.Action == actionCharge {
		lp.log.DEBUG.Printf("optimizer: charge (%.0fW)", s.Charge)
		return lp.fastCharging()
	}

	// minpv keeps its minimum power, the optimizer plans with it
	if mode == api.ModeMinPV {
		lp.log.DEBUG.Println("optimizer: stop, keeping minimum power")
		return lp.minCharging()
	}

	lp.log.DEBUG.Println("optimizer: stop")
	return lp.setLimit(0)
}
