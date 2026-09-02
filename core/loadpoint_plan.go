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

	pc := precondition.String()
	if precondition >= 7*24*time.Hour {
		pc = "everything" // 168h, UI sentinel for max
	}

	lp.log.TRACE.Printf("plan: creating plan with continuous=%v, precondition=%s, duration=%v, target=%v",
		continuous, pc, requiredDuration.Round(time.Second), targetTime.Round(time.Second).Local())

	return lp.planner.Plan(requiredDuration, precondition, targetTime, continuous)
}

// planConstraints are the inputs a charge plan is computed from. Gathering them
// has no side effects on loadpoint state.
type planConstraints struct {
	connected        bool             // a vehicle is connected
	time             time.Time        // effective plan time
	goal             float64          // soc in % or energy in kWh
	socBased         bool             // goal is soc, not energy
	maxPower         float64          // effective max charge power
	requiredDuration time.Duration    // estimated duration to reach the goal
	strategy         api.PlanStrategy // precondition, continuous
}

// planConstraints gathers the current plan constraints. None of the reads have
// side effects.
func (lp *Loadpoint) planConstraints() planConstraints {
	var c planConstraints

	c.connected = lp.connected()
	c.time = lp.EffectivePlanTime()
	c.goal, c.socBased = lp.GetPlanGoal()
	c.maxPower = lp.EffectiveMaxPower()
	c.requiredDuration = lp.GetPlanRequiredDuration(c.goal, c.maxPower)
	c.strategy = lp.getEffectivePlanStrategy()

	return c
}

// planEvaluation records one evaluation of the plan constraints: the constraints
// it was given and the plan they produced
type planEvaluation struct {
	// constraints go in
	constraints planConstraints

	// plan comes out
	plan       api.Rates
	start, end time.Time
	overrun    time.Duration
}

// evaluatePlan gathers the plan constraints and computes the resulting charge plan.
// It has no side effects on loadpoint state and may run outside the control cycle.
func (lp *Loadpoint) evaluatePlan() planEvaluation {
	pe := planEvaluation{constraints: lp.planConstraints()}

	if !pe.constraints.connected || pe.constraints.time.IsZero() {
		return pe
	}

	// expired plans are only kept alive while active
	if lp.clock.Until(pe.constraints.time) < 0 && (!lp.planActive || !pe.constraints.connected) {
		return pe
	}

	if pe.constraints.requiredDuration <= 0 {
		return pe
	}

	pe.plan = lp.GetPlan(pe.constraints.time, pe.constraints.requiredDuration, pe.constraints.strategy.Precondition, pe.constraints.strategy.Continuous)
	if pe.plan == nil {
		return pe
	}

	pe.start = planner.Start(pe.plan)
	pe.end = planner.End(pe.plan)

	if excessDuration := pe.constraints.requiredDuration - lp.clock.Until(pe.constraints.time); excessDuration > 0 {
		pe.overrun = excessDuration
	}

	return pe
}

// publishPlanState publishes the given plan state
func (lp *Loadpoint) publishPlanState(pe planEvaluation) {
	lp.publish(keys.Plan, pe.plan)
	lp.publish(keys.PlanProjectedStart, pe.start)
	lp.publish(keys.PlanProjectedEnd, pe.end)
	lp.publish(keys.PlanOverrun, pe.overrun)
}

// publishPlan computes and publishes the charge plan. Unlike plannerActive it has
// no side effects, so it runs for every loadpoint on every control cycle rather
// than only for the loadpoint the cycle is updating.
func (lp *Loadpoint) publishPlan() {
	lp.publishPlanState(lp.evaluatePlan())
}

// plannerActive checks if the charging plan has a currently active slot
func (lp *Loadpoint) plannerActive() (active bool) {
	defer func() {
		lp.setPlanActive(active)
	}()

	pe := lp.evaluatePlan()

	defer func() {
		lp.publishPlanState(pe)
	}()

	// re-check since plannerActive() is called before connected() check in Update()
	if !pe.constraints.connected {
		return false
	}

	if pe.constraints.time.IsZero() {
		lp.log.DEBUG.Println("!! plan: plan time zero")
		return false
	}

	// keep overrunning plans as long as a vehicle is connected
	if lp.clock.Until(pe.constraints.time) < 0 && (!lp.planActive || !pe.constraints.connected) {
		lp.log.DEBUG.Println("plan: deleting expired plan")
		lp.finishPlan()
		return false
	}

	if pe.constraints.requiredDuration <= 0 {
		// continue a 100% plan as long as the vehicle is connected
		if lp.planActive && pe.constraints.socBased && pe.constraints.goal == 100 {
			return true
		}
		lp.log.DEBUG.Println("!! plan: required duration 0")

		lp.finishPlan()
		return false
	}

	if pe.plan == nil {
		lp.log.DEBUG.Println("!! plan: plan nil")
		return false
	}

	var overrun string
	if pe.overrun > 0 {
		overrun = fmt.Sprintf("overruns by %v, ", pe.overrun.Round(time.Second))
		if !lp.planOverrunSent {
			lp.pushEvent("planoverrun")
			lp.planOverrunSent = true
		}
	}

	lp.log.DEBUG.Printf("plan: charge %v between %v until %v (%spower: %.0fW, avg cost: %.3f)",
		planner.Duration(pe.plan).Round(time.Second), pe.start.Round(time.Second).Local(), pe.constraints.time.Round(time.Second).Local(), overrun,
		pe.constraints.maxPower, planner.AverageCost(pe.plan))

	// log plan
	for _, slot := range pe.plan {
		lp.log.TRACE.Printf("  slot from: %v to %v cost %.3f", slot.Start.Round(time.Second).Local(), slot.End.Round(time.Second).Local(), slot.Value)
	}

	activeSlot := planner.SlotAt(lp.clock.Now(), pe.plan)
	active = !activeSlot.End.IsZero()

	if active {
		// ignore short plans if not already active
		if slotRemaining := lp.clock.Until(activeSlot.End); !lp.planActive && slotRemaining < tariff.SlotDuration-time.Minute && !planner.SlotHasSuccessor(activeSlot, pe.plan) {
			lp.log.DEBUG.Printf("plan: slot too short- ignoring remaining %v", slotRemaining.Round(time.Second))
			return false
		}

		// lock the goal when soc-based plan becomes active for the first time
		if lp.planLocked.Id == 0 && pe.constraints.socBased {
			lp.lockPlanGoal(pe.constraints.time, int(pe.constraints.goal), lp.getPlanId())
		}

		// remember last active plan's slot end time
		lp.planSlotEnd = activeSlot.End
	} else if lp.planActive {
		// planner was active (any slot, not necessarily previous slot) and charge goal has not yet been met
		switch {
		case lp.clock.Now().After(pe.constraints.time) && !pe.constraints.time.IsZero():
			// if the plan did not (entirely) work, we may still be charging beyond plan end- in that case, continue charging
			// TODO check when schedule is implemented
			lp.log.DEBUG.Println("plan: continuing after target time")
			return true
		case lp.clock.Now().Before(lp.planSlotEnd) && !lp.planSlotEnd.IsZero() && pe.constraints.requiredDuration > pe.constraints.strategy.Precondition:
			// don't stop an already running slot if goal was not met
			lp.log.DEBUG.Printf("plan: continuing until end of slot at %s", lp.planSlotEnd.Round(time.Second).Local())
			return true
		case pe.constraints.requiredDuration < tariff.SlotDuration && pe.constraints.requiredDuration > pe.constraints.strategy.Precondition:
			lp.log.DEBUG.Printf("plan: continuing for remaining %v", pe.constraints.requiredDuration.Round(time.Second))
			return true
		case lp.clock.Until(pe.start) < tariff.SlotDuration-time.Minute:
			lp.log.DEBUG.Printf("plan: avoid re-start within %v, continuing for remaining %v", tariff.SlotDuration, lp.clock.Until(pe.start).Round(time.Second))
			return true
		case pe.constraints.strategy.Continuous && pe.constraints.requiredDuration > pe.constraints.strategy.Precondition:
			lp.log.DEBUG.Printf("plan: ignoring restart at %s for continuous charging", pe.start.Round(time.Second).Local())
			pe.start = lp.clock.Now()
			pe.end = pe.start.Add(pe.constraints.requiredDuration)
			return true
		}
	}

	return active
}
