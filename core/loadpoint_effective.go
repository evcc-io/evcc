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

	vehicleSoc := lp.vehicleSoc
	if chargedSoc, ok := lp.unsampledSoc(lp.getVehicleAndSocEstimator()); ok {
		vehicleSoc = chargedSoc
	}

	for _, p := range plans {
		if vehicleSoc == 0 || vehicleSoc < float64(p.Soc) {
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

// vehiclePlans returns the static plan and each occurrence of active repeating plans within the given days
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

// GetSocPlanGoals returns the soc goals of all vehicle plans, repeating plans for each occurrence
// within a week. Unlike nextVehiclePlan it includes later and already reached plans.
func (lp *Loadpoint) GetSocPlanGoals() []loadpoint.PlanGoal {
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

// currentLimits holds the effective min and max current per number of active phases
type currentLimits struct {
	min, max [3]float64
}

// limitIndex maps active phases to the limits index, unknown phases count as 3p
func limitIndex(phases int) int {
	return min(max(expect(phases), 1), 3) - 1
}

// minCurrent returns the effective min current for the given active phases
func (l currentLimits) minCurrent(phases int) float64 {
	return l.min[limitIndex(phases)]
}

// maxCurrent returns the effective max current for the given active phases
func (l currentLimits) maxCurrent(phases int) float64 {
	return l.max[limitIndex(phases)]
}

// uniformCurrentLimits returns limits that do not depend on the active phases
func uniformCurrentLimits(minCurrent, maxCurrent float64) currentLimits {
	return currentLimits{
		min: [3]float64{minCurrent, minCurrent, minCurrent},
		max: [3]float64{maxCurrent, maxCurrent, maxCurrent},
	}
}

// effectiveCurrentLimits returns the effective min and max current for 1, 2 and 3 active phases
func (lp *Loadpoint) effectiveCurrentLimits() currentLimits {
	lpMin, lpMax := lp.getMinCurrent(), lp.getMaxCurrent()
	var vehicleMin, vehicleMax, chargerMin, chargerMax float64

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			vehicleMin = res
		}
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok {
			vehicleMax = res
		}
	}

	if c, ok := api.Cap[api.CurrentLimiter](lp.charger); ok {
		if minCurrent, maxCurrent, err := c.GetMinMaxCurrent(); err == nil {
			chargerMin, chargerMax = minCurrent, maxCurrent
		}
	}

	// power-limited chargers (e.g. EEBus OHPCF heat pump) report their demand in W
	var minPower, maxPower float64
	if c, ok := api.Cap[api.PowerLimiter](lp.charger); ok {
		if minP, maxP, err := c.GetMinMaxPower(); err == nil {
			minPower, maxPower = minP, maxP
		}
	}

	phaseLimiter, hasPhaseLimiter := api.Cap[api.PhaseCurrentLimiter](lp.charger)

	// coarse chargers truncate to full amps in setLimit, so round derived limits up to keep
	// the enable gate reachable and a fixed power request (min == max) consistent (#31549)
	derived := func(current float64) float64 {
		if lp.coarseCurrent() {
			return math.Ceil(current)
		}
		return current
	}

	var res currentLimits
	for i := range res.min {
		phases := i + 1
		phaseMin, phaseMax := chargerMin, chargerMax

		if hasPhaseLimiter {
			if minCurrent, maxCurrent, err := phaseLimiter.GetPhaseMinMaxCurrent(phases); err == nil {
				if minCurrent > 0 {
					phaseMin = max(phaseMin, derived(minCurrent))
				}
				if maxCurrent > 0 {
					phaseMax = minPositive(phaseMax, derived(maxCurrent))
				}
			}
		}

		if minPower > 0 {
			phaseMin = max(phaseMin, derived(grid.PowerToCurrent(minPower, phases)))
		}
		if maxPower > 0 {
			phaseMax = minPositive(phaseMax, derived(grid.PowerToCurrent(maxPower, phases)))
		}

		switch {
		case max(vehicleMin, phaseMin) == 0:
			res.min[i] = lpMin
		case phaseMin > 0:
			res.min[i] = max(vehicleMin, phaseMin)
		default:
			res.min[i] = max(vehicleMin, lpMin)
		}

		res.max[i] = lpMax
		for _, limit := range []float64{vehicleMax, phaseMax} {
			if limit > 0 {
				res.max[i] = min(res.max[i], limit)
			}
		}
	}

	return res
}

// minPositive returns the smaller of both values, ignoring unset (non-positive) values
func minPositive(a, b float64) float64 {
	switch {
	case b <= 0:
		return a
	case a <= 0:
		return b
	default:
		return min(a, b)
	}
}

// limitPhases returns the active phases the current limits apply to without logging
// phase mismatches. Non-switching chargers keep the maximum active phases.
func (lp *Loadpoint) limitPhases() int {
	if lp.hasPhaseSwitching() {
		return min(expect(lp.phases), lp.maxActivePhases())
	}
	return lp.maxActivePhases()
}

// uniform reports whether the limits do not depend on the active phases
func (l currentLimits) uniform() bool {
	return l.min[0] == l.min[1] && l.min[1] == l.min[2] && l.max[0] == l.max[1] && l.max[1] == l.max[2]
}

// activeCurrentLimits returns the effective min and max current for the active phases
func (lp *Loadpoint) activeCurrentLimits() (float64, float64) {
	limits := lp.effectiveCurrentLimits()
	if limits.uniform() {
		return limits.min[0], limits.max[0]
	}
	phases := lp.limitPhases()
	return limits.minCurrent(phases), limits.maxCurrent(phases)
}

// effectiveMinCurrent returns the effective min current for the active phases
func (lp *Loadpoint) effectiveMinCurrent() float64 {
	res, _ := lp.activeCurrentLimits()
	return res
}

// effectiveMaxCurrent returns the effective max current for the active phases
func (lp *Loadpoint) effectiveMaxCurrent() float64 {
	_, res := lp.activeCurrentLimits()
	return res
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
	phases := lp.minActivePhases()
	return grid.CurrentToPower(lp.effectiveCurrentLimits().minCurrent(phases), phases)
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
	phases := lp.maxActivePhases()
	res := grid.CurrentToPower(lp.effectiveCurrentLimits().maxCurrent(phases), phases)
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
