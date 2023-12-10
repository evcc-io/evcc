package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/vehicle"
)

const (
	smallSlotDuration = 10 * time.Minute // small planner slot duration we might ignore
	smallGapDuration  = 60 * time.Minute // small gap duration between planner slots we might ignore
)

// TODO planActive is not guarded by mutex

// setPlanActive updates plan active flag
func (lp *Loadpoint) setPlanActive(active bool) {
	if !active {
		lp.planSlotEnd = time.Time{}
	}
	if lp.planActive != active {
		lp.planActive = active
		lp.publish(keys.PlanActive, lp.planActive)
	}
}

// deletePlan deletes the charging plan, either loadpoint or vehicle
func (lp *Loadpoint) deletePlan() {
	lp.setPlanEnergy(time.Time{}, 0)

	if v := lp.GetVehicle(); v != nil {
		vehicle.Settings(lp.log, v).SetPlanSoc(time.Time{}, 0)
	}
}

// remainingPlanEnergy returns missing energy amount in kWh
func (lp *Loadpoint) remainingPlanEnergy() (float64, bool) {
	_, limit := lp.GetPlanEnergy()
	return max(0, limit-lp.getChargedEnergy()/1e3),
		limit > 0 && !lp.socBasedPlanning()
}

// planRequiredDuration is the estimated total charging duration
func (lp *Loadpoint) planRequiredDuration(maxPower float64) time.Duration {
	if energy, ok := lp.remainingPlanEnergy(); ok {
		return time.Duration(energy * 1e3 / maxPower * float64(time.Hour))
	}

	v := lp.GetVehicle()
	if v == nil || lp.socEstimator == nil {
		return 0
	}

	_, soc := vehicle.Settings(lp.log, v).GetPlanSoc()

	return lp.socEstimator.RemainingChargeDuration(soc, maxPower)
}

// GetPlan creates a charging plan
//
// Results:
// - required total charging duration
// - actual charging plan as rate table
func (lp *Loadpoint) GetPlan(targetTime time.Time, maxPower float64) (time.Duration, api.Rates, error) {
	if lp.planner == nil || targetTime.IsZero() {
		return 0, nil, nil
	}

	requiredDuration := lp.planRequiredDuration(maxPower)
	if requiredDuration <= 0 {
		return 0, nil, nil
	}

	plan, err := lp.planner.Plan(requiredDuration, targetTime)

	return requiredDuration, plan, err
}

// plannerActive checks if the charging plan has a currently active slot
func (lp *Loadpoint) plannerActive() (active bool) {
	defer func() {
		lp.setPlanActive(active)
	}()

	var planStart time.Time
	defer func() {
		lp.publish(keys.PlanProjectedStart, planStart)
	}()

	maxPower := lp.EffectiveMaxPower()
	planTime := lp.EffectivePlanTime()

	requiredDuration, plan, err := lp.GetPlan(planTime, maxPower)
	if err != nil {
		lp.log.ERROR.Println("planner:", err)
		return false
	}

	// nothing to do now-invalid plan from the past
	if requiredDuration == 0 || (lp.clock.Until(planTime) < 0 && !lp.planActive) {
		lp.deletePlan()
		return false
	}

	var overrun string
	if excessDuration := requiredDuration - lp.clock.Until(planTime); excessDuration > 0 {
		overrun = fmt.Sprintf("overruns by %v, ", excessDuration.Round(time.Second))
	}

	planStart = planner.Start(plan)
	lp.log.DEBUG.Printf("plan: charge %v starting at %v until %v (%spower: %.0fW, avg cost: %.3f)",
		planner.Duration(plan).Round(time.Second), planStart.Round(time.Second).Local(), planTime.Round(time.Second).Local(), overrun,
		maxPower, planner.AverageCost(plan))

	// log plan
	for _, slot := range plan {
		lp.log.TRACE.Printf("  slot from: %v to %v cost %.3f", slot.Start.Round(time.Second).Local(), slot.End.Round(time.Second).Local(), slot.Price)
	}

	activeSlot := planner.SlotAt(lp.clock.Now(), plan)
	active = !activeSlot.End.IsZero()

	if active {
		// ignore short plans if not already active
		if slotRemaining := lp.clock.Until(activeSlot.End); !lp.planActive && slotRemaining < smallSlotDuration && !planner.SlotHasSuccessor(activeSlot, plan) {
			lp.log.DEBUG.Printf("plan: slot too short- ignoring remaining %v", slotRemaining.Round(time.Second))
			return false
		}

		// remember last active plan's end time
		lp.setPlanActive(true)
		lp.planSlotEnd = activeSlot.End
	} else if lp.planActive {
		// planner was active (any slot, not necessarily previous slot) and charge goal has not yet been met
		switch {
		case lp.clock.Now().After(planTime) && !planTime.IsZero():
			// if the plan did not (entirely) work, we may still be charging beyond plan end- in that case, continue charging
			// TODO check when schedule is implemented
			lp.log.DEBUG.Println("plan: continuing after target time")
			return true
		case lp.clock.Now().Before(lp.planSlotEnd) && !lp.planSlotEnd.IsZero():
			// don't stop an already running slot if goal was not met
			lp.log.DEBUG.Println("plan: continuing until end of slot")
			return true
		case requiredDuration < smallGapDuration:
			lp.log.DEBUG.Printf("plan: continuing for remaining %v", requiredDuration.Round(time.Second))
			return true
		case lp.clock.Until(planStart) < smallGapDuration:
			lp.log.DEBUG.Printf("plan: will re-start shortly, continuing for remaining %v", lp.clock.Until(planStart).Round(time.Second))
			return true
		}
	}

	return active
}
