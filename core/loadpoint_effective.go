package core

import (
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/samber/lo"
)

// PublishEffectiveValues publishes all effective values
func (lp *Loadpoint) PublishEffectiveValues() {
	lp.publish(keys.EffectivePriority, lp.EffectivePriority())
	lp.publish(keys.EffectivePlanId, lp.EffectivePlanId())
	lp.publish(keys.EffectivePlanTime, lp.EffectivePlanTime())
	lp.publish(keys.EffectivePlanSoc, lp.EffectivePlanSoc())
	lp.publish(keys.EffectivePlanStrategy, lp.EffectivePlanStrategy())
	lp.publish(keys.EffectiveMinCurrent, lp.effectiveMinCurrent())
	lp.publish(keys.EffectiveMaxCurrent, lp.effectiveMaxCurrent())
	lp.publish(keys.EffectiveMinSoc, lp.EffectiveMinSoc())
	lp.publish(keys.EffectiveLimitSoc, lp.EffectiveLimitSoc())
}

// EffectivePriority returns the effective priority
func (lp *Loadpoint) EffectivePriority() int {
	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetPriority(); ok {
			return res
		}
	}
	return lp.GetPriority()
}

type plan struct {
	Id    int
	Start time.Time // last possible start time
	End   time.Time // user-selected finish time
	Soc   int
}

func (lp *Loadpoint) nextActivePlan(maxPower float64, plans []plan) *plan {
	for i, p := range plans {
		requiredDuration := lp.getPlanRequiredDuration(float64(p.Soc), maxPower)
		plans[i].Start = p.End.Add(-requiredDuration)
	}

	// sort plans by start time
	slices.SortStableFunc(plans, func(i, j plan) int {
		return i.Start.Compare(j.Start)
	})

	for _, p := range plans {
		if lp.vehicleSoc == 0 || lp.vehicleSoc < float64(p.Soc) {
			return &p
		}
	}

	return nil
}

// nextVehiclePlan returns the next vehicle plan time, soc, id
// Returns locked plan if available, otherwise calculates fresh
func (lp *Loadpoint) nextVehiclePlan() (time.Time, int, int) {
	// return locked plan if available
	if p := lp.planLocked; p.Id > 0 {
		return p.Time, p.Soc, p.Id
	}

	// calculate fresh plan
	if v := lp.GetVehicle(); v != nil {
		// calculate earliest required plan start
		if plan := lp.nextActivePlan(lp.effectiveMaxPower(), lp.vehiclePlans(v, 1)); plan != nil {
			return plan.End, plan.Soc, plan.Id
		}
	}
	return time.Time{}, 0, 0
}

// vehiclePlans returns the static plan and the occurrences of active repeating plans within the next days
func (lp *Loadpoint) vehiclePlans(v api.Vehicle, days int) []plan {
	var plans []plan

	// static plan
	if planTime, soc := vehicle.Settings(lp.log, v).GetPlanSoc(); soc != 0 {
		plans = append(plans, plan{Id: 1, Soc: soc, End: planTime})
	}

	// repeating plans
	for index, rp := range vehicle.Settings(lp.log, v).GetRepeatingPlans() {
		if !rp.Active || len(rp.Weekdays) == 0 {
			continue
		}

		planTime, err := util.GetNextOccurrence(rp.Weekdays, rp.Time, rp.Tz)
		if err != nil {
			lp.log.DEBUG.Printf("invalid repeating plan: weekdays=%v, time=%s, tz=%s, error=%v", rp.Weekdays, rp.Time, rp.Tz, err)
			continue
		}

		for d := range days {
			if ts := planTime.AddDate(0, 0, d); slices.Contains(rp.Weekdays, int(ts.Weekday())) {
				plans = append(plans, plan{Id: index + 2, Soc: rp.Soc, End: ts})
			}
		}
	}

	return plans
}

// GetPlanGoals returns the soc goals of all vehicle plans within the coming week,
// repeating plans once per occurrence. Unlike nextVehiclePlan it includes later
// and already reached plans, so the optimizer can plan for all of them at once.
func (lp *Loadpoint) GetPlanGoals() []loadpoint.PlanGoal {
	lp.RLock()
	defer lp.RUnlock()

	v := lp.GetVehicle()
	if v == nil || !lp.socBasedPlanning() {
		return nil
	}

	return lo.Map(lp.vehiclePlans(v, 7), func(p plan, _ int) loadpoint.PlanGoal {
		return loadpoint.PlanGoal{Time: p.End, Soc: p.Soc}
	})
}

// EffectivePlanSoc returns the soc target for the current plan
func (lp *Loadpoint) EffectivePlanSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	_, soc, _ := lp.nextVehiclePlan()
	return soc
}

// getPlanId returns the plan id of the current/next plan
func (lp *Loadpoint) getPlanId() int {
	if lp.socBasedPlanning() {
		_, _, id := lp.nextVehiclePlan()
		return id
	}
	if lp.planEnergy > 0 {
		return 1
	}
	return 0
}

// EffectivePlanId returns the id for the current plan
func (lp *Loadpoint) EffectivePlanId() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getPlanId()
}

// EffectivePlanTime returns the effective plan time
func (lp *Loadpoint) EffectivePlanTime() time.Time {
	lp.RLock()
	defer lp.RUnlock()
	if lp.socBasedPlanning() {
		ts, _, _ := lp.nextVehiclePlan()
		return ts
	}

	ts, _ := lp.getPlanEnergy()
	return ts
}

// SocBasedPlanning returns true if soc based planning is enabled
func (lp *Loadpoint) SocBasedPlanning() bool {
	return lp.socBasedPlanning()
}

// effectiveMinCurrent returns the effective min current
func (lp *Loadpoint) effectiveMinCurrent() float64 {
	lpMin := lp.getMinCurrent()
	var vehicleMin, chargerMin float64

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			vehicleMin = res
		}
	}

	if c, ok := api.Cap[api.CurrentLimiter](lp.charger); ok {
		if res, _, err := c.GetMinMaxCurrent(); err == nil {
			chargerMin = res
		}
	}

	// power-limited chargers (e.g. EEBus OHPCF heat pump) report their demand in
	// W; convert to per-phase current so the PV enable gate covers it
	if c, ok := api.Cap[api.PowerLimiter](lp.charger); ok {
		if res, _, err := c.GetMinMaxPower(); err == nil && res > 0 {
			chargerMin = grid.PowerToCurrent(res, lp.minActivePhases())
			// coarse chargers truncate to full amps in setLimit, so round the
			// demand up to keep the enable gate reachable (#31549)
			if lp.coarseCurrent() {
				chargerMin = math.Ceil(chargerMin)
			}
		}
	}

	switch {
	case max(vehicleMin, chargerMin) == 0:
		return lpMin
	case chargerMin > 0:
		return max(vehicleMin, chargerMin)
	default:
		return max(vehicleMin, lpMin)
	}
}

// effectiveMaxCurrent returns the effective max current
func (lp *Loadpoint) effectiveMaxCurrent() float64 {
	maxCurrent := lp.getMaxCurrent()

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	if c, ok := api.Cap[api.CurrentLimiter](lp.charger); ok {
		if _, res, err := c.GetMinMaxCurrent(); err == nil && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	if c, ok := api.Cap[api.PowerLimiter](lp.charger); ok {
		if _, res, err := c.GetMinMaxPower(); err == nil && res > 0 {
			powerMax := grid.PowerToCurrent(res, lp.maxActivePhases())
			// match effectiveMinCurrent's rounding so a fixed power request
			// (min == max) doesn't yield min > max on coarse chargers (#31549)
			if lp.coarseCurrent() {
				powerMax = math.Ceil(powerMax)
			}
			maxCurrent = min(maxCurrent, powerMax)
		}
	}

	return maxCurrent
}

// EffectiveMinSoc returns the effective min soc (heating: min temperature)
func (lp *Loadpoint) EffectiveMinSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.effectiveMinSoc()
}

// effectiveMinSoc returns the effective min soc (heating: min temperature)
func (lp *Loadpoint) effectiveMinSoc() int {
	minSoc := lp.minSoc

	// loadpoint and vehicle min soc are independent limits- honor both
	if v := lp.GetVehicle(); v != nil {
		minSoc = max(minSoc, vehicle.Settings(lp.log, v).GetMinSoc())
	}

	return minSoc
}

// EffectiveLimitSoc returns the effective session limit soc
func (lp *Loadpoint) EffectiveLimitSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.effectiveLimitSoc()
}

// effectiveLimitSoc returns the effective session limit soc
// TODO take vehicle api limits into account
func (lp *Loadpoint) effectiveLimitSoc() int {
	if lp.limitSoc > 0 {
		return lp.limitSoc
	}

	if v := lp.GetVehicle(); v != nil {
		if soc := vehicle.Settings(lp.log, v).GetLimitSoc(); soc > 0 {
			return soc
		}
	}

	// MUST return 100 here as UI looks at effectiveLimitSoc and not limitSoc (VehicleSoc.vue)
	return 100
}

// EffectiveStepPower returns the effective step power for the currently active phases
func (lp *Loadpoint) EffectiveStepPower() float64 {
	return grid.CurrentToPower(1, lp.ActivePhases())
}

// EffectiveMinPower returns the effective min power for the minimum active phases
func (lp *Loadpoint) EffectiveMinPower() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return grid.CurrentToPower(lp.effectiveMinCurrent(), lp.minActivePhases())
}

// EffectiveMaxPower returns the effective max power taking vehicle capabilities,
// phase scaling and load management power limits into account
func (lp *Loadpoint) EffectiveMaxPower() float64 {
	lp.RLock()
	defer lp.RUnlock()

	if circuitMaxPower := circuitMaxPower(lp.circuit); circuitMaxPower > 0 {
		return min(lp.effectiveMaxPower(), circuitMaxPower)
	}

	return lp.effectiveMaxPower()
}

// effectiveMaxPower returns the effective max power taking vehicle capabilities and phase scaling into account
func (lp *Loadpoint) effectiveMaxPower() float64 {
	res := grid.CurrentToPower(lp.effectiveMaxCurrent(), lp.maxActivePhases())
	if lp.vehicle != nil {
		if maxPower, ok := lp.vehicle.OnIdentified().GetMaxPower(); ok {
			return min(maxPower, res)
		}
	}
	return res
}

// EffectivePlanStrategy returns the effective plan strategy
func (lp *Loadpoint) EffectivePlanStrategy() api.PlanStrategy {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getEffectivePlanStrategy()
}

func (lp *Loadpoint) getEffectivePlanStrategy() api.PlanStrategy {
	if v := lp.GetVehicle(); v != nil {
		if lp.socBasedPlanning() {
			return vehicle.Settings(lp.log, v).GetPlanStrategy()
		}
	}

	return lp.getPlanStrategy()
}
