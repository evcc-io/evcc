package soc

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	deviation = 30 * time.Minute
)

// Timer is the target charging handler
type Timer struct {
	Adapter
	log          *util.Logger
	current      float64
	projectedEnd time.Time
	active       bool
	validated    bool
}

// NewTimer creates a Timer
func NewTimer(log *util.Logger, api Adapter) *Timer {
	lp := &Timer{
		log:     log,
		Adapter: api,
	}

	return lp
}

// MustValidateDemand resets the flag for detecting if DemandActive has been called
func (lp *Timer) MustValidateDemand() {
	lp.validated = false
}

// DemandValidated returns if DemandActive has been called
func (lp *Timer) DemandValidated() bool {
	return lp.validated
}

// Stop stops the target charging request
func (lp *Timer) Stop() {
	if lp.active {
		lp.active = false
		lp.Publish("targetTimeActive", lp.active)
		lp.log.DEBUG.Println("target charging: disable")
	}
}

// Reset resets the target charging request
func (lp *Timer) Reset() {
	lp.SetTargetTime(time.Time{})
	lp.Stop()
}

// DemandActive calculates remaining charge duration and returns true if charge start is required to achieve target soc in time
func (lp *Timer) DemandActive() bool {
	if lp.GetTargetTime().IsZero() {
		return false
	}

	// demand validation has been called
	lp.validated = true

	// power
	power := lp.GetMaxPower()
	if lp.active {
		power *= lp.current / lp.GetMaxCurrent()
	}

	se := lp.SocEstimator()
	if se == nil {
		lp.log.WARN.Println("target charging: not possible")
		return false
	}

	// time
	targetSoC := lp.GetTargetSoC()
	targetTime := lp.GetTargetTime()

	remainingDuration := time.Duration(float64(se.AssumedChargeDuration(targetSoC, power)) / chargeEfficiency)
	lp.projectedEnd = time.Now().Add(remainingDuration).Round(time.Minute)

	lp.log.DEBUG.Printf("estimated charge duration: %v to %d%% at %.0fW", remainingDuration.Round(time.Minute), targetSoC, power)
	if lp.active {
		lp.log.DEBUG.Printf("projected end: %v", lp.projectedEnd)
		lp.log.DEBUG.Printf("desired finish time: %v", targetTime)
	} else {
		lp.log.DEBUG.Printf("projected start: %v", targetTime.Add(-remainingDuration))
	}

	// timer charging is already active- only deactivate once charging has stopped
	if lp.active {
		if time.Now().After(targetTime) && lp.GetStatus() != api.StatusC {
			lp.Stop()
		}

		return lp.active
	}

	// check if charging need be activated
	if active := lp.projectedEnd.After(targetTime); active {
		lp.active = active
		lp.Publish("targetTimeActive", lp.active)

		lp.current = lp.GetMaxCurrent()
		lp.log.INFO.Printf("target charging active for %v: projected %v (%v remaining)", targetTime, lp.projectedEnd, remainingDuration.Round(time.Minute))
	}

	return lp.active
}

// Handle adjusts current up/down to achieve desired target time taking.
func (lp *Timer) Handle() float64 {
	action := "steady"

	targetTime := lp.GetTargetTime()

	switch {
	case lp.projectedEnd.Before(targetTime.Add(-deviation)):
		lp.current--
		action = "slowdown"

	case lp.projectedEnd.After(targetTime):
		lp.current++
		action = "speedup"
	}

	lp.current = math.Max(math.Min(lp.current, lp.GetMaxCurrent()), lp.GetMinCurrent())
	lp.log.DEBUG.Printf("target charging: %s (%.3gA)", action, lp.current)

	return lp.current
}
