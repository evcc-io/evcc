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

// Reset resets the target charging request
func (lp *Timer) Reset() {
	if lp == nil {
		return
	}

	lp.current = lp.GetMaxCurrent()
	lp.Time = time.Time{}
	lp.active = false
}

// DemandActive calculates remaining charge duration and returns true if charge start is required to achieve target soc in time
func (lp *Timer) DemandActive() bool {
	if lp == nil {
		return false
	}

	if lp.Time.IsZero() {
		lp.active = false
		return false
	}

	defer func() {
		lp.Publish("timerSet", lp.Time.After(time.Now()))
		lp.Publish("timerActive", lp.active)
		lp.Publish("timerProjectedEnd", lp.finishAt)
	}()

	// power
	power := lp.GetMaxPower()
	if lp.active {
		power *= lp.current / lp.GetMaxCurrent()
	}

	se := lp.SocEstimator()
	if se == nil {
		lp.log.WARN.Printf("target charging: not possible")
		return false
	}

	// time
	remainingDuration := se.AssumedChargeDuration(lp.SoC, power)
	lp.finishAt = time.Now().Add(remainingDuration).Round(time.Minute)

	lp.log.DEBUG.Printf("estimated charge power: %vW", power)
	lp.log.DEBUG.Printf("estimated charge duration: %v to %v%%", remainingDuration.Round(time.Minute), lp.SoC)
	lp.log.DEBUG.Printf("projected start: %v", lp.Time.Add(-remainingDuration))
	if lp.active {
		lp.log.DEBUG.Printf("projected end: %v", lp.finishAt)
		lp.log.DEBUG.Printf("desired finish time: %v", lp.Time)
	}

	// timer charging is already active- only deactivate once charging has stopped
	if lp.active {
		if time.Now().After(lp.Time) && lp.GetStatus() != api.StatusC {
			lp.log.INFO.Printf("target charging: deactivating")
			lp.active = false
		}

		return lp.active
	}

	// check if charging need be activated
	if lp.active = lp.finishAt.After(lp.Time); lp.active {
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
