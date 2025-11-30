package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// checkSmartLimit checks if current rate meets smart limit and returns next start time if not active.
// checkBelow: true for rate <= limit, false for rate >= limit
func (lp *Loadpoint) checkSmartLimit(limit *float64, rates api.Rates, checkBelow bool) (bool, time.Time) {
	var nextStart time.Time

	active := lp.smartLimitActive(limit, rates, checkBelow)
	if !active {
		nextStart = lp.smartLimitNextStart(limit, rates, checkBelow)
	}

	return active, nextStart
}

func (lp *Loadpoint) smartLimitActive(limit *float64, rates api.Rates, checkBelow bool) bool {
	rate, err := rates.At(time.Now())
	if err != nil || limit == nil {
		return false
	}

	if checkBelow {
		return rate.Value <= *limit
	}
	return rate.Value >= *limit
}

// smartLimitNextStart returns the next start time when the smart limit condition will be met
func (lp *Loadpoint) smartLimitNextStart(limit *float64, rates api.Rates, checkBelow bool) time.Time {
	if limit == nil || rates == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rates {
		if slot.Start.After(now) && (checkBelow && slot.Value <= *limit || !checkBelow && slot.Value >= *limit) {
			return slot.Start
		}
	}

	return time.Time{}
}
