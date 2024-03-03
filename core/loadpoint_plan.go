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
	if !lp.socBasedPlanning() {
		lp.setPlanEnergy(time.Time{}, 0)
	} else if v := lp.GetVehicle(); v != nil {
		vehicle.Settings(lp.log, v).SetPlanSoc(time.Time{}, 0)
	}
}

// remainingPlanEnergy returns missing energy amount in kWh
func (lp *Loadpoint) remainingPlanEnergy(planEnergy float64) float64 {
	return max(0, planEnergy-lp.getChargedEnergy()/1e3)
}

// GetPlanRequiredDuration is the estimated total charging duration
func (lp *Loadpoint) GetPlanRequiredDuration(goal, maxPower float64) time.Duration {
	lp.RLock()
	defer lp.RUnlock()

	if lp.socBasedPlanning() {
		if lp.socEstimator == nil {
			return 0
		}
		return lp.socEstimator.RemainingChargeDuration(int(goal), maxPower)
	}

	energy := lp.remainingPlanEnergy(goal)
	return time.Duration(energy * 1e3 / maxPower * float64(time.Hour))
}

// GetPlanGoal returns the plan goal in %, true or kWh, false
func (lp *Loadpoint) GetPlanGoal() (float64, bool) {
	lp.RLock()
	defer lp.RUnlock()

	if lp.socBasedPlanning() {
		_, soc := vehicle.Settings(lp.log, lp.GetVehicle()).GetPlanSoc()
		return float64(soc), true
	}

	_, limit := lp.GetPlanEnergy()
	return limit, false
}

// GetPlan creates a charging plan for given time and duration
func (lp *Loadpoint) GetPlan(targetTime time.Time, requiredDuration time.Duration) (api.Rates, error) {
	if lp.planner == nil || targetTime.IsZero() {
		return nil, nil
	}

	return lp.planner.Plan(requiredDuration, targetTime)
}

// plannerActive checks if the charging plan has a currently active slot
func (lp *Loadpoint) plannerActive() (active bool) {
	defer func() {
		lp.setPlanActive(active)
	}()

	var planStart time.Time
	var planOverrun bool
	defer func() {
		lp.publish(keys.PlanProjectedStart, planStart)
		lp.publish(keys.PlanOverrun, planOverrun)
	}()

	planTime := lp.EffectivePlanTime()
	if planTime.IsZero() {
		return false
	}
	if lp.clock.Until(planTime) < 0 && !lp.planActive {
		lp.deletePlan()
		return false
	}

	goal, isSocBased := lp.GetPlanGoal()
	maxPower := lp.EffectiveMaxPower()
	requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
	if requiredDuration <= 0 {
		// continue a 100% plan as long as the vehicle is charging
		if lp.planActive && isSocBased && goal == 100 && lp.charging() {
			return true
		}

		lp.deletePlan()
		return false
	}

	plan, err := lp.GetPlan(planTime, requiredDuration)
	if err != nil {
		lp.log.ERROR.Println("planner:", err)
		return false
	}

	var overrun string
	if excessDuration := requiredDuration - lp.clock.Until(planTime); excessDuration > 0 {
		overrun = fmt.Sprintf("overruns by %v, ", excessDuration.Round(time.Second))
		planOverrun = true
	}

	planStart = planner.Start(plan)
	lp.log.DEBUG.Printf("plan: charge %v between %v until %v (%spower: %.0fW, avg cost: %.3f)",
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
			lp.log.DEBUG.Printf("plan: avoid re-start within %v, continuing for remaining %v", smallGapDuration, lp.clock.Until(planStart).Round(time.Second))
			return true
		}
	}

	return active
}
