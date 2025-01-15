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
	lp.publish(keys.EffectivePriority, lp.EffectivePriority())
	lp.publish(keys.EffectivePlanId, lp.EffectivePlanId())
	lp.publish(keys.EffectivePlanTime, lp.EffectivePlanTime())
	lp.publish(keys.EffectivePlanSoc, lp.EffectivePlanSoc())
	lp.publish(keys.EffectiveMinCurrent, lp.effectiveMinCurrent())
	lp.publish(keys.EffectiveMaxCurrent, lp.effectiveMaxCurrent())
	lp.publish(keys.EffectiveLimitSoc, lp.effectiveLimitSoc())
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
		requiredDuration := lp.GetPlanRequiredDuration(float64(p.Soc), maxPower)
		plans[i].Start = p.End.Add(-requiredDuration)
	}

	// sort plans by start time
	slices.SortStableFunc(plans, func(i, j plan) int {
		return i.Start.Compare(j.Start)
	})

	if len(plans) > 0 {
		return &plans[0]
	}

	return nil
}

// nextVehiclePlan returns the next vehicle plan time, soc and id
func (lp *Loadpoint) nextVehiclePlan() (time.Time, int, int) {
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

			time, err := util.GetNextOccurrence(rp.Weekdays, rp.Time, rp.Tz)
			if err != nil {
				lp.log.DEBUG.Printf("invalid repeating plan: weekdays=%v, time=%s, tz=%s, error=%v", rp.Weekdays, rp.Time, rp.Tz, err)
				continue
			}

			plans = append(plans, plan{Id: index + 2, Soc: rp.Soc, End: time})
		}

		// calculate earliest required plan start
		if plan := lp.nextActivePlan(lp.EffectiveMaxPower(), plans); plan != nil {
			return plan.End, plan.Soc, plan.Id
		}
	}
	return time.Time{}, 0, 0
}

// EffectivePlanSoc returns the soc target for the current plan
func (lp *Loadpoint) EffectivePlanSoc() int {
	_, soc, _ := lp.nextVehiclePlan()
	return soc
}

// EffectivePlanId returns the id for the current plan
func (lp *Loadpoint) EffectivePlanId() int {
	if lp.socBasedPlanning() {
		_, _, id := lp.nextVehiclePlan()
		return id
	}
	if lp.planEnergy > 0 {
		return 1
	}
	// no plan
	return 0
}

// EffectivePlanTime returns the effective plan time
func (lp *Loadpoint) EffectivePlanTime() time.Time {
	if lp.socBasedPlanning() {
		ts, _, _ := lp.nextVehiclePlan()
		return ts
	}

	ts, _ := lp.GetPlanEnergy()
	return ts
}

// SocBasedPlanning returns true if soc based planning is enabled
func (lp *Loadpoint) SocBasedPlanning() bool {
	return lp.socBasedPlanning()
}

// effectiveMinCurrent returns the effective min current
func (lp *Loadpoint) effectiveMinCurrent() float64 {
	lpMin := lp.GetMinCurrent()
	var vehicleMin, chargerMin float64

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			vehicleMin = res
		}
	}

	if c, ok := lp.charger.(api.CurrentLimiter); ok {
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
	maxCurrent := lp.GetMaxCurrent()

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	if c, ok := lp.charger.(api.CurrentLimiter); ok {
		if _, res, err := c.GetMinMaxCurrent(); err == nil && res > 0 {
			maxCurrent = min(maxCurrent, res)
		}
	}

	return maxCurrent
}

// effectiveLimitSoc returns the effective session limit soc
// TODO take vehicle api limits into account
func (lp *Loadpoint) effectiveLimitSoc() int {
	lp.RLock()
	defer lp.RUnlock()

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

// effectiveStepPower returns the effective step power for the currently active phases
func (lp *Loadpoint) effectiveStepPower() float64 {
	return Voltage * float64(lp.ActivePhases())
}

// EffectiveMinPower returns the effective min power for the minimum active phases
func (lp *Loadpoint) EffectiveMinPower() float64 {
	return Voltage * lp.effectiveMinCurrent() * float64(lp.minActivePhases())
}

// EffectiveMaxPower returns the effective max power taking vehicle capabilities and phase scaling into account
func (lp *Loadpoint) EffectiveMaxPower() float64 {
	return Voltage * lp.effectiveMaxCurrent() * float64(lp.maxActivePhases())
}
