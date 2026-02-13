package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/tariff"
)

// TODO planActive is not guarded by mutex

// PlanLock contains information about a locked plan
type PlanLock struct {
	Time time.Time // target time (committed goal, persists during overrun)
	Soc  int       // target soc
	Id   int       // id (0=none, 1=static, 2+=repeating), needed to highlight the plan in ui
}

// clearPlanLock clears the locked plan goal
func (lp *Loadpoint) clearPlanLock() {
	lp.planLocked = PlanLock{}
}

// ClearPlanLock clears the locked plan goal
func (lp *Loadpoint) ClearPlanLock() {
	lp.Lock()
	defer lp.Unlock()
	lp.clearPlanLock()
}

// lockPlanGoal locks the current plan goal to handle overruns (soc-based plans)
func (lp *Loadpoint) lockPlanGoal(planTime time.Time, soc int, id int) {
	lp.planLocked = PlanLock{
		Time: planTime,
		Soc:  soc,
		Id:   id,
	}
}

// setPlanActive updates plan active flag
func (lp *Loadpoint) setPlanActive(active bool) {
	if !active {
		lp.planOverrunSent = false
		lp.planSlotEnd = time.Time{}
		lp.clearPlanLock()
	}
	if lp.planActive != active {
		lp.planActive = active
		lp.publish(keys.PlanActive, lp.planActive)
	}
}

// finishPlan deletes the charging plan, either loadpoint or vehicle
func (lp *Loadpoint) finishPlan() {
	if lp.repeatingPlanning() {
		return // noting to do
	} else if !lp.socBasedPlanning() {
		lp.setPlanEnergy(time.Time{}, 0)
	} else if v := lp.GetVehicle(); v != nil {
		vehicle.Settings(lp.log, v).SetPlanSoc(time.Time{}, 0)
	}
}

// remainingPlanEnergy returns missing energy amount in kWh
func (lp *Loadpoint) remainingPlanEnergy(planEnergy float64) float64 {
	return max(0, planEnergy-(lp.getChargedEnergy()/1e3-lp.planEnergyOffset))
}

// GetPlanRequiredDuration is the estimated total charging duration
func (lp *Loadpoint) GetPlanRequiredDuration(goal, maxPower float64) time.Duration {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getPlanRequiredDuration(goal, maxPower)
}

// getPlanRequiredDuration is the estimated total charging duration
func (lp *Loadpoint) getPlanRequiredDuration(goal, maxPower float64) time.Duration {
	if lp.socBasedPlanning() {
		if lp.socEstimator == nil {
			return soc.RemainingChargeDuration(goal, maxPower, lp.vehicleSoc, lp.GetVehicle().Capacity())
		}
		return lp.socEstimator.RemainingChargeDuration(goal, maxPower)
	}

	energy := lp.remainingPlanEnergy(goal)
	return time.Duration(energy * 1e3 / maxPower * float64(time.Hour))
}

// GetPlanGoal returns the plan goal in %, true or kWh, false
func (lp *Loadpoint) GetPlanGoal() (float64, bool) {
	lp.RLock()
	defer lp.RUnlock()

	if lp.socBasedPlanning() {
		_, soc, _ := lp.nextVehiclePlan()
		return float64(soc), true
	}

	_, limit := lp.getPlanEnergy()
	return limit, false
}

// GetPlan creates a charging plan for given time and duration
// The plan is sorted by time
func (lp *Loadpoint) GetPlan(targetTime time.Time, requiredDuration, precondition time.Duration, continuous bool) api.Rates {
	if lp.planner == nil || targetTime.IsZero() {
		return nil
	}

	lp.log.TRACE.Printf("plan: creating plan with continuous=%v, precondition=%v, duration=%v, target=%v",
		continuous, precondition, requiredDuration.Round(time.Second), targetTime.Round(time.Second).Local())

	return lp.planner.Plan(requiredDuration, precondition, targetTime, continuous)
}

// plannerActive checks if the charging plan has a currently active slot
func (lp *Loadpoint) plannerActive() (active bool) {
	defer func() {
		lp.setPlanActive(active)
	}()

	var plan api.Rates
	var planStart, planEnd time.Time
	var planOverrun time.Duration

	defer func() {
		lp.publish(keys.Plan, plan)
		lp.publish(keys.PlanProjectedStart, planStart)
		lp.publish(keys.PlanProjectedEnd, planEnd)
		lp.publish(keys.PlanOverrun, planOverrun)
	}()

	// re-check since plannerActive() is called before connected() check in Update()
	if !lp.connected() {
		return false
	}

	planTime := lp.EffectivePlanTime()
	if planTime.IsZero() {
		lp.log.DEBUG.Println("!! plan: plan time zero")
		return false
	}

	// keep overrunning plans as long as a vehicle is connected
	if lp.clock.Until(planTime) < 0 && (!lp.planActive || !lp.connected()) {
		lp.log.DEBUG.Println("plan: deleting expired plan")
		lp.finishPlan()
		return false
	}

	goal, isSocBased := lp.GetPlanGoal()
	maxPower := lp.EffectiveMaxPower()
	requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
	if requiredDuration <= 0 {
		// continue a 100% plan as long as the vehicle is connected
		if lp.planActive && isSocBased && goal == 100 {
			return true
		}
		lp.log.DEBUG.Println("!! plan: required duration 0")

		lp.finishPlan()
		return false
	}

	strategy := lp.getEffectivePlanStrategy()

	plan = lp.GetPlan(planTime, requiredDuration, strategy.Precondition, strategy.Continuous)
	if plan == nil {
		lp.log.DEBUG.Println("!! plan: plan nil")
		return false
	}

	var overrun string
	if excessDuration := requiredDuration - lp.clock.Until(planTime); excessDuration > 0 {
		overrun = fmt.Sprintf("overruns by %v, ", excessDuration.Round(time.Second))
		planOverrun = excessDuration
		if !lp.planOverrunSent {
			lp.pushEvent("planoverrun")
			lp.planOverrunSent = true
		}
	}

	planStart = planner.Start(plan)
	planEnd = planner.End(plan)
	lp.log.DEBUG.Printf("plan: charge %v between %v until %v (%spower: %.0fW, avg cost: %.3f)",
		planner.Duration(plan).Round(time.Second), planStart.Round(time.Second).Local(), planTime.Round(time.Second).Local(), overrun,
		maxPower, planner.AverageCost(plan))

	// log plan
	for _, slot := range plan {
		lp.log.TRACE.Printf("  slot from: %v to %v cost %.3f", slot.Start.Round(time.Second).Local(), slot.End.Round(time.Second).Local(), slot.Value)
	}

	activeSlot := planner.SlotAt(lp.clock.Now(), plan)
	active = !activeSlot.End.IsZero()

	if active {
		// ignore short plans if not already active
		if slotRemaining := lp.clock.Until(activeSlot.End); !lp.planActive && slotRemaining < tariff.SlotDuration-time.Minute && !planner.SlotHasSuccessor(activeSlot, plan) {
			lp.log.DEBUG.Printf("plan: slot too short- ignoring remaining %v", slotRemaining.Round(time.Second))
			return false
		}

		// lock the goal when soc-based plan becomes active for the first time
		if lp.planLocked.Id == 0 && isSocBased {
			lp.lockPlanGoal(planTime, int(goal), lp.getPlanId())
		}

		// remember last active plan's slot end time
		lp.planSlotEnd = activeSlot.End
	} else if lp.planActive {
		// planner was active (any slot, not necessarily previous slot) and charge goal has not yet been met
		switch {
		case lp.clock.Now().After(planTime) && !planTime.IsZero():
			// if the plan did not (entirely) work, we may still be charging beyond plan end- in that case, continue charging
			// TODO check when schedule is implemented
			lp.log.DEBUG.Println("plan: continuing after target time")
			return true
		case lp.clock.Now().Before(lp.planSlotEnd) && !lp.planSlotEnd.IsZero() && requiredDuration > strategy.Precondition:
			// don't stop an already running slot if goal was not met
			lp.log.DEBUG.Printf("plan: continuing until end of slot at %s", lp.planSlotEnd.Round(time.Second).Local())
			return true
		case requiredDuration < tariff.SlotDuration && requiredDuration > strategy.Precondition:
			lp.log.DEBUG.Printf("plan: continuing for remaining %v", requiredDuration.Round(time.Second))
			return true
		case lp.clock.Until(planStart) < tariff.SlotDuration-time.Minute:
			lp.log.DEBUG.Printf("plan: avoid re-start within %v, continuing for remaining %v", tariff.SlotDuration, lp.clock.Until(planStart).Round(time.Second))
			return true
		case strategy.Continuous && requiredDuration > strategy.Precondition:
			lp.log.DEBUG.Printf("plan: ignoring restart at %s for continuous charging", planStart.Round(time.Second).Local())
			return true
		}
	}

	return active
}
