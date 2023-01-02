package core

import (
	"time"

	"github.com/evcc-io/evcc/core/soc"
)

// setPlanActive updates plan active flag
func (lp *Loadpoint) setPlanActive(active bool) {
	if !active {
		lp.planSlotEnd = time.Time{}
	}
	lp.planActive = active
	lp.publish("planActive", lp.planActive)
}

// plannerActive checks if charging plan is active
func (lp *Loadpoint) plannerActive() (active bool) {
	defer func() {
		lp.publish(targetTimeActive, active)
	}()

	if lp.planner == nil || lp.socEstimator == nil || lp.targetTime.IsZero() {
		return false
	}

	targetSoc := 100
	maxPower := lp.GetMaxPower()
	var requiredDuration time.Duration

	if energy, ok := lp.remainingChargeEnergy(); ok {
		if energy > 0 {
			requiredDuration = time.Duration(energy * 1e3 / maxPower * float64(time.Hour))
		}
	} else {
		// TODO vehicle soc limit
		if lp.Soc.target > 0 {
			targetSoc = lp.Soc.target
		}
		requiredDuration = lp.socEstimator.RemainingChargeDuration(targetSoc, maxPower)
	}
	requiredDuration = time.Duration(float64(requiredDuration) / soc.ChargeEfficiency)

	// anticipate lower charge rates at end of charging curve
	if targetSoc >= 80 {
		requiredDuration = time.Duration(float64(requiredDuration) / soc.ChargeEfficiency)

		if targetSoc >= 90 {
			requiredDuration = time.Duration(float64(requiredDuration) / soc.ChargeEfficiency)
		}
	}

	lp.log.DEBUG.Printf("planning %v until %v at %.0fW", requiredDuration.Round(time.Second), lp.targetTime.Round(time.Second).Local(), maxPower)

	planStart, slotEnd, active, err := lp.planner.Active(requiredDuration, lp.targetTime)
	if err != nil {
		lp.log.ERROR.Println("planner:", err)
		return false
	}
	lp.publish(targetTimeProjectedStart, planStart)

	if active {
		// ignore short plans if not already active
		if !lp.planActive && requiredDuration < 10*time.Minute {
			lp.log.DEBUG.Printf("plan too short- ignoring remaining %v", requiredDuration.Round(time.Second))
			return false
		}

		// remember last active plan's end time
		lp.setPlanActive(true)
		lp.planSlotEnd = slotEnd
	} else if lp.planActive {
		// planner was active (any slot, not necessarily previous slot) and charge goal has not yet been met
		switch {
		case lp.clock.Now().After(lp.targetTime) && !lp.targetTime.IsZero():
			// if the plan did not (entirely) work, we may still be charging beyond plan end- in that case, continue charging
			// TODO check when schedule is implemented
			lp.log.DEBUG.Println("continuing after target time")
			active = true
		case lp.clock.Now().Before(lp.planSlotEnd) && !lp.planSlotEnd.IsZero():
			// don't stop an already running slot if goal was not met
			lp.log.DEBUG.Println("continuing until end of slot")
			active = true
		case requiredDuration < 30*time.Minute:
			lp.log.DEBUG.Printf("continuing for remaining %v", requiredDuration.Round(time.Second))
			active = true
		}
	}

	return active
}
