package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// TODO: @andig this is a blind copy if loadpoint_smartcost.go to the the correct behaviour
// refactor to avoid redundancy

func (lp *Loadpoint) updateFeedinPriority(limit *float64, rates api.Rates) (bool, time.Time) {
	var nextStart time.Time

	active := lp.feedinPriorityActive(limit, rates)
	if !active {
		nextStart = lp.feedinPriorityNextStart(limit, rates)
	}

	return active, nextStart
}

func (lp *Loadpoint) feedinPriorityActive(limit *float64, rates api.Rates) bool {
	rate, err := rates.At(time.Now())
	return err == nil && limit != nil && rate.Value >= *limit
}

// feedinPriorityNextStart returns the next start time for a smart feedin priority zone
func (lp *Loadpoint) feedinPriorityNextStart(limit *float64, rates api.Rates) time.Time {
	if limit == nil || rates == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rates {
		if slot.Start.After(now) && slot.Value >= *limit {
			return slot.Start
		}
	}

	return time.Time{}
}
