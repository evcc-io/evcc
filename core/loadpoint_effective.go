package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
)

// PublishEffectiveValues publishes all effective values
func (lp *Loadpoint) PublishEffectiveValues() {
	lp.publish(keys.EffectivePriority, lp.effectivePriority())
	lp.publish(keys.EffectivePriorityScore, lp.EffectivePriorityScore(lp.GetPriorityBasis()))
	lp.publish(keys.EffectivePlanId, lp.EffectivePlanId())
	lp.publish(keys.EffectivePlanTime, lp.EffectivePlanTime())
	lp.publish(keys.EffectivePlanSoc, lp.EffectivePlanSoc())
	lp.publish(keys.EffectivePlanStrategy, lp.EffectivePlanStrategy())
	lp.publish(keys.EffectiveMinCurrent, lp.effectiveMinCurrent())
	lp.publish(keys.EffectiveMaxCurrent, lp.effectiveMaxCurrent())
	lp.publish(keys.EffectiveLimitSoc, lp.EffectiveLimitSoc())
}

// effectivePriority returns the effective priority tier (integer part of the score).
func (lp *Loadpoint) effectivePriority() int {
	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetPriority(); ok {
			return res
		}
	}
	return lp.GetPriority()
}

// EffectivePriorityScore ranks loadpoints for surplus distribution: the integer part is
// the effective priority tier, the fractional part [0,1) sub-orders within the tier by the
// priority strategy/basis (higher wins). Basis is passed in so the prioritizer can score a
// whole tier on one scale, see Prioritizer.effectiveBasis.
func (lp *Loadpoint) EffectivePriorityScore(basis api.PriorityBasis) float64 {
	score := float64(lp.effectivePriority())

	soc := lp.GetSoc()
	if soc <= 0 {
		return score
	}

	// gap is the soc-% quantity the strategy ranks by (a larger gap scores higher)
	var gap float64
	switch lp.GetPriorityStrategy() {
	case api.PrioritySoc:
		gap = 100 - soc
	case api.PriorityDeficit:
		gap = float64(lp.EffectiveLimitSoc()) - soc
	default:
		return score
	}

	// energy basis: convert the soc-% gap into absolute kWh using the vehicle
	// capacity, falling back to the percentage gap when capacity is unknown
	if basis == api.PriorityBasisEnergy {
		if capacity := lp.vehicleCapacity(); capacity > 0 {
			gap = gap / 100 * capacity
		} else {
			lp.log.DEBUG.Println("priority basis energy: unknown vehicle capacity, ranking by soc percentage")
		}
	}

	return score + priorityFraction(gap)
}

// vehicleCapacity returns the active vehicle's usable capacity in kWh, or 0 if
// no vehicle is active or its capacity is unknown.
func (lp *Loadpoint) vehicleCapacity() float64 {
	if v := lp.GetVehicle(); v != nil {
		return v.Capacity()
	}
	return 0
}

// priorityFraction maps a soc-based value to a [0,1) sub-ordering offset, kept
// strictly below 1 so it can never bump a loadpoint into the next priority tier.
func priorityFraction(v float64) float64 {
	switch {
	case v <= 0:
		return 0
	case v > 99:
		return 0.99
	default:
		return v / 100
	}
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

			plans = append(plans, plan{Id: index + 2, Soc: rp.Soc, End: planTime})
		}

		// calculate earliest required plan start
		if plan := lp.nextActivePlan(lp.effectiveMaxPower(), plans); plan != nil {
			return plan.End, plan.Soc, plan.Id
		}
	}
	return time.Time{}, 0, 0
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

	return maxCurrent
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
	return Voltage * float64(lp.ActivePhases())
}

// EffectiveMinPower returns the effective min power for the minimum active phases
func (lp *Loadpoint) EffectiveMinPower() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return Voltage * lp.effectiveMinCurrent() * float64(lp.minActivePhases())
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
	res := Voltage * lp.effectiveMaxCurrent() * float64(lp.maxActivePhases())
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
