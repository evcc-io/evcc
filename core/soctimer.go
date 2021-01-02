package core

import (
	"math"
	"time"
)

const (
	utilization float64 = 0.6
	deviation           = 30 * time.Minute
)

// SoCTimer is the target charging handler
type SoCTimer struct {
	*LoadPoint
	SoC            int
	Time           time.Time
	finishAt       time.Time
	chargeRequired bool
}

// Supported returns true if target charging is possible, i.e. the vehicle soc can be determined
func (lp *SoCTimer) Supported() bool {
	return lp.socEstimator != nil
}

// Reset resets the target charging request
func (lp *SoCTimer) Reset() {
	if lp != nil {
		lp.Time = time.Time{}
		lp.SoC = 0
	}
}

// active returns true if there is an active target charging request
func (lp *SoCTimer) active() bool {
	if lp == nil {
		return false
	}

	inactive := lp.Time.IsZero() || lp.Time.Before(time.Now())
	lp.publish("socTimerSet", !inactive)

	// reset active
	if inactive && lp.chargeRequired {
		lp.chargeRequired = false
		lp.publish("socTimerActive", lp.chargeRequired)
	}

	return !inactive
}

// StartRequired calculates remaining charge duration and returns true if charge start is required to achieve target soc in time
func (lp *SoCTimer) StartRequired() bool {
	if !lp.active() {
		return false
	}

	current := lp.effectiveCurrent()

	// use start current for calculation if currently not charging
	if current == 0 {
		current = float64(lp.MaxCurrent) * utilization
		current = math.Max(math.Min(current, float64(lp.MaxCurrent)), 0)
	}

	power := current * float64(lp.Phases) * Voltage

	// time
	remainingDuration := lp.socEstimator.RemainingChargeDuration(power, lp.SoC)
	lp.finishAt = time.Now().Add(remainingDuration).Round(time.Minute)
	lp.log.DEBUG.Printf("target charging active for %v: projected %v (%v remaining)", lp.Time, lp.finishAt, remainingDuration.Round(time.Minute))

	lp.chargeRequired = lp.finishAt.After(lp.Time)
	lp.publish("socTimerActive", lp.chargeRequired)

	return lp.chargeRequired
}

// Handle adjusts current up/down to achieve desired target time taking.
// PV mode target current into consideration to ensure maximum PV usage.
func (lp *SoCTimer) Handle(pvCurrent float64) error {
	current := lp.maxCurrent

	switch {
	case lp.finishAt.Before(lp.Time.Add(-deviation)):
		current--
		lp.log.DEBUG.Printf("target charging: slowdown")

	case lp.finishAt.After(lp.Time):
		current++
		lp.log.DEBUG.Printf("target charging: speedup")
	}

	// use higher-charging pv current if available
	if current < pvCurrent {
		current = pvCurrent
	}

	current = math.Max(math.Min(current, float64(lp.MaxCurrent)), 0)

	return lp.setLimit(current, false)
}
