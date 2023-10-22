package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

// publishEffectiveValues publishes all effective values
func (lp *Loadpoint) publishEffectiveValues() {
	lp.publish(keys.EffectivePriority, lp.effectivePriority())
	lp.publish(keys.EffectiveMinCurrent, lp.effectiveMinCurrent())
	lp.publish(keys.EffectiveMaxCurrent, lp.effectiveMaxCurrent())
	lp.publish(keys.EffectiveLimitSoc, lp.effectiveLimitSoc())
}

// effectivePriority returns the effective priority
func (lp *Loadpoint) effectivePriority() int {
	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetPriority(); ok {
			return res
		}
	}
	return lp.GetPriority()
}

// effectiveMinCurrent returns the effective min current
func (lp *Loadpoint) effectiveMinCurrent() float64 {
	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMinCurrent(); ok {
			return res
		}
	}

	if c, ok := lp.charger.(api.CurrentLimiter); ok {
		if res, _, err := c.GetMinMaxCurrent(); err == nil {
			return res
		}
	}

	return lp.GetMinCurrent()
}

// effectiveMaxCurrent returns the effective max current
func (lp *Loadpoint) effectiveMaxCurrent() float64 {
	if v := lp.GetVehicle(); v != nil {
		if res, ok := v.OnIdentified().GetMaxCurrent(); ok {
			return res
		}
	}

	if c, ok := lp.charger.(api.CurrentLimiter); ok {
		if _, res, err := c.GetMinMaxCurrent(); err == nil {
			return res
		}
	}

	return lp.GetMaxCurrent()
}

// effectiveLimitSoc returns the effective session limit soc.
// Session takes precedence over vehicle.
// TODO take vehicle api limits into account
func (lp *Loadpoint) effectiveLimitSoc() int {
	lp.RLock()
	defer lp.RUnlock()

	if lp.limitSoc > 0 {
		return lp.limitSoc
	}

	if v := lp.GetVehicle(); v != nil {
		if soc, ok := v.OnIdentified().GetLimitSoc(); ok {
			return soc
		}
	}

	return 100
}
