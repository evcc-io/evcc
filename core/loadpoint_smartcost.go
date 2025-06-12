package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

func (lp *Loadpoint) updateSmartCost(limit *float64, rates api.Rates) (bool, time.Time) {
	var nextStart time.Time

	active := lp.smartCostActive(limit, rates)
	if !active {
		nextStart = lp.smartCostNextStart(limit, rates)
	}

	return active, nextStart
}

func (lp *Loadpoint) smartCostActive(limit *float64, rates api.Rates) bool {
	rate, err := rates.At(time.Now())
	return err == nil && limit != nil && rate.Value <= *limit
}

// smartCostNextStart returns the next start time for a smart cost rate below the limit
func (lp *Loadpoint) smartCostNextStart(limit *float64, rates api.Rates) time.Time {
	if limit == nil || rates == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rates {
		if slot.Start.After(now) && slot.Value <= *limit {
			return slot.Start
		}
	}

	return time.Time{}
}
