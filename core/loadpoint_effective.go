package core

import (
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
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
	Id      int
	Start   time.Time // last possible start time
	End     time.Time // user-selected finish time
	Soc     int       // user-selected (or absence-derived) target soc
	Goal    int       // planning goal including accumulated absence soc drops, capped at 100
	Absence *api.PlanAbsence
}

func (lp *Loadpoint) nextActivePlan(maxPower float64, plans []plan) *plan {
	for i, p := range plans {
		requiredDuration := lp.getPlanRequiredDuration(float64(p.Goal), maxPower)
		plans[i].Start = p.End.Add(-requiredDuration)
	}

	// sort plans by start time
	slices.SortStableFunc(plans, func(i, j plan) int {
		return i.Start.Compare(j.Start)
	})

	for _, p := range plans {
		if lp.vehicleSoc == 0 || lp.vehicleSoc < float64(p.Goal) {
			return &p
		}
	}

	return nil
}

// vehiclePlans returns all vehicle plans sorted by target time. Absence plans without
// explicit soc get a derived soc covering the drop plus min soc. Accumulated soc drops
// of earlier absences increase the planning goal of later plans.
func (lp *Loadpoint) vehiclePlans() []plan {
	v := lp.GetVehicle()
	if v == nil {
		return nil
	}

	var plans []plan

	// static plan
	if planTime, soc, absence := vehicle.Settings(lp.log, v).GetPlanSoc(); soc != 0 || absence != nil {
		plans = append(plans, plan{Id: 1, Soc: soc, End: planTime, Absence: absence})
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

		plans = append(plans, plan{Id: index + 2, Soc: rp.Soc, End: planTime, Absence: rp.Absence})
	}

	// sort plans by target time
	slices.SortStableFunc(plans, func(i, j plan) int {
		return i.End.Compare(j.End)
	})

	minSoc := lp.effectiveMinSoc()
	var drops int
	for i := range plans {
		p := &plans[i]
		if p.Soc == 0 && p.Absence != nil {
			p.Soc = min(100, minSoc+p.Absence.Soc)
		}
		p.Goal = min(100, p.Soc+drops)
		if p.Absence != nil {
			drops += p.Absence.Soc
		}
	}

	return plans
}

// nextVehiclePlan returns the next vehicle plan
// Returns locked plan if available, otherwise calculates fresh
func (lp *Loadpoint) nextVehiclePlan() *plan {
	// return locked plan if available
	if p := lp.planLocked; p.Id > 0 {
		return &plan{Id: p.Id, End: p.Time, Soc: p.Soc, Goal: p.Goal}
	}

	// calculate earliest required plan start
	return lp.nextActivePlan(lp.effectiveMaxPower(), lp.vehiclePlans())
}

// EffectivePlanSoc returns the soc target for the current plan
func (lp *Loadpoint) EffectivePlanSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	if p := lp.nextVehiclePlan(); p != nil {
		return p.Soc
	}
	return 0
}

// getPlanId returns the plan id of the current/next plan
func (lp *Loadpoint) getPlanId() int {
	if lp.socBasedPlanning() {
		if p := lp.nextVehiclePlan(); p != nil {
			return p.Id
		}
		return 0
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
		if p := lp.nextVehiclePlan(); p != nil {
			return p.End
		}
		return time.Time{}
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
			chargerMin = res / (Voltage * float64(lp.minActivePhases()))
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
			powerMax := res / (Voltage * float64(lp.maxActivePhases()))
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
