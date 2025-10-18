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
	Id           int
	Start        time.Time // last possible start time
	End          time.Time // user-selected finish time
	Precondition time.Duration
	Soc          int
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

// NextVehiclePlan returns the next vehicle plan time, soc and id
func (lp *Loadpoint) NextVehiclePlan() (time.Time, time.Duration, int, int) {
	lp.RLock()
	defer lp.RUnlock()
	return lp.nextVehiclePlan()
}

// nextVehiclePlan returns the next vehicle plan time, precondition duration, soc and id
func (lp *Loadpoint) nextVehiclePlan() (time.Time, time.Duration, int, int) {
	if v := lp.GetVehicle(); v != nil {
		var plans []plan

		// static plan
		if planTime, precondition, soc := vehicle.Settings(lp.log, v).GetPlanSoc(); soc != 0 {
			plans = append(plans, plan{Id: 1, Precondition: precondition, Soc: soc, End: planTime})
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

			precondition := time.Duration(rp.Precondition) * time.Second
			plans = append(plans, plan{Id: index + 2, Precondition: precondition, Soc: rp.Soc, End: planTime})
		}

		// calculate earliest required plan start
		if plan := lp.nextActivePlan(lp.effectiveMaxPower(), plans); plan != nil {
			return plan.End, plan.Precondition, plan.Soc, plan.Id
		}
	}
	return time.Time{}, 0, 0, 0
}

// EffectivePlanSoc returns the soc target for the current plan
func (lp *Loadpoint) EffectivePlanSoc() int {
	_, _, soc, _ := lp.NextVehiclePlan()
	return soc
}

// EffectivePlanId returns the id for the current plan
func (lp *Loadpoint) EffectivePlanId() int {
	if lp.socBasedPlanning() {
		_, _, _, id := lp.NextVehiclePlan()
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
		ts, _, _, _ := lp.NextVehiclePlan()
		return ts
	}

	ts, _, _ := lp.GetPlanEnergy()
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
	maxCurrent := lp.getMaxCurrent()

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
	return Voltage * lp.effectiveMaxCurrent() * float64(lp.maxActivePhases())
}
