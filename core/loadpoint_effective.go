package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/vehicle"
)

// PublishEffectiveValues publishes all effective values
func (lp *Loadpoint) PublishEffectiveValues() {
	lp.publish(keys.EffectivePriority, lp.EffectivePriority())
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

// vehiclePlanSoc returns the next vehicle plan time and soc
func (lp *Loadpoint) vehiclePlanSoc() (time.Time, int) {
	if v := lp.GetVehicle(); v != nil {
		return vehicle.Settings(lp.log, v).GetPlanSoc()
	}
	return time.Time{}, 0
}

// EffectivePlanSoc returns the soc target for the current plan
func (lp *Loadpoint) EffectivePlanSoc() int {
	_, soc := lp.vehiclePlanSoc()
	return soc
}

// EffectivePlanTime returns the effective plan time
func (lp *Loadpoint) EffectivePlanTime() time.Time {
	if lp.socBasedPlanning() {
		ts, _ := lp.vehiclePlanSoc()
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
	// There are multiple modes:
	// a) The user has defined a minCurrent for the LP, the max value of all sources wins
	// b) The user has not defined a minCurrent for the loadpoint
	//	  This will use the default 6A, and only a higher value can override this
	//	  OR a minCurrent provided by the charger

	lpMinCurrent := lp.GetMinCurrent()
	minCurrent := lpMinCurrent

	if c, ok := lp.charger.(api.CurrentLimiter); ok {
		if chargerMinCurrent, _, err := c.GetMinMaxCurrent(); err == nil {
			if lp.HasUserDefinedMinCurrent() {
				// Case a)
				minCurrent = max(minCurrent, chargerMinCurrent)
			} else {
				// Case b)
				minCurrent = chargerMinCurrent
			}
		}
	}

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			minCurrent = max(minCurrent, res)
		}
	}

	if lpMinCurrent != minCurrent {
		lp.publish(keys.EffectiveMinCurrent, minCurrent)
	}

	return minCurrent
}

// effectiveMaxCurrent returns the effective max current
func (lp *Loadpoint) effectiveMaxCurrent() float64 {
	maxCurrent := lp.GetMaxCurrent()

	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok {
			maxCurrent = min(maxCurrent, res)
		}
	}

	if c, ok := lp.charger.(api.CurrentLimiter); ok {
		if _, res, err := c.GetMinMaxCurrent(); err == nil {
			maxCurrent = min(maxCurrent, res)
			lp.publish(keys.EffectiveMaxCurrent, maxCurrent)
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

// EffectiveMinPower returns the effective min power for a single phase
func (lp *Loadpoint) EffectiveMinPower() float64 {
	// TODO check if 1p available
	return Voltage * lp.effectiveMinCurrent()
}

// EffectiveMaxPower returns the effective max power taking vehicle capabilities and phase scaling into account
func (lp *Loadpoint) EffectiveMaxPower() float64 {
	return Voltage * lp.effectiveMaxCurrent() * float64(lp.maxActivePhases())
}
