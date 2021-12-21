package soc

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	chargeEfficiency = 0.8
	deviation        = 30 * time.Minute
)

// Timer is the target charging handler
type Timer struct {
	Adapter
	log      *util.Logger
	current  float64
	SoC      int
	Time     time.Time
	finishAt time.Time
	active   bool
}

// NewTimer creates a Timer
func NewTimer(log *util.Logger, api Adapter) *Timer {
	lp := &Timer{
		log:     log,
		Adapter: api,
	}

	return lp
}

// Stop stops the target charging request
func (lp *Timer) Stop() {
	if lp == nil {
		return
	}

	if lp.active {
		lp.active = false
		lp.Publish("targetTimeActive", lp.active)
		lp.log.DEBUG.Println("target charging: disable")
	}
}

// Reset resets the target charging request
func (lp *Timer) Reset() {
	if lp == nil {
		return
	}

	lp.current = lp.GetMaxCurrent()
	lp.Time = time.Time{}

	lp.active = false
	lp.Publish("targetTimeActive", lp.active)
}

// DemandActive calculates remaining charge duration and returns true if charge start is required to achieve target soc in time
func (lp *Timer) DemandActive() bool {
	if lp == nil || lp.Time.IsZero() {
		return false
	}

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
	remainingDuration := time.Duration(float64(se.AssumedChargeDuration(lp.SoC, power)) / chargeEfficiency)
	lp.finishAt = time.Now().Add(remainingDuration).Round(time.Minute)

	lp.log.DEBUG.Printf("estimated charge duration: %v to %d%% at %.0fW", remainingDuration.Round(time.Minute), lp.SoC, power)
	if lp.active {
		lp.log.DEBUG.Printf("projected end: %v", lp.finishAt)
		lp.log.DEBUG.Printf("desired finish time: %v", lp.Time)
	} else {
		lp.log.DEBUG.Printf("projected start: %v", lp.Time.Add(-remainingDuration))
	}

	// timer charging is already active- only deactivate once charging has stopped
	if lp.active {
		if time.Now().After(lp.Time) && lp.GetStatus() != api.StatusC {
			lp.Stop()
		}

		return lp.active
	}

	// check if charging need be activated
	if active := lp.finishAt.After(lp.Time); active {
		lp.active = active
		lp.Publish("targetTimeActive", lp.active)

		lp.current = lp.GetMaxCurrent()
		lp.log.INFO.Printf("target charging active for %v: projected %v (%v remaining)", lp.Time, lp.finishAt, remainingDuration.Round(time.Minute))
	}

	return lp.active
}

// Handle adjusts current up/down to achieve desired target time taking.
func (lp *Timer) Handle() float64 {
	action := "steady"

	switch {
	case lp.finishAt.Before(lp.Time.Add(-deviation)):
		lp.current--
		action = "slowdown"

	case lp.finishAt.After(lp.Time):
		lp.current++
		action = "speedup"
	}

	lp.current = math.Max(math.Min(lp.current, lp.GetMaxCurrent()), lp.GetMinCurrent())
	lp.log.DEBUG.Printf("target charging: %s (%.3gA)", action, lp.current)

	return lp.current
}
