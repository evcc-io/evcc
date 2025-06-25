package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// NOTE: scale is either 1.0 or -1.0 where the latter indicates inverting the comparison
func (lp *Loadpoint) updateSmartCost(limit *float64, rates api.Rates, scale float64) (bool, time.Time) {
	var nextStart time.Time

	active := lp.smartCostActive(limit, rates, scale)
	if !active {
		nextStart = lp.smartCostNextStart(limit, rates, scale)
	}

	return active, nextStart
}

func (lp *Loadpoint) smartCostActive(limit *float64, rates api.Rates, scale float64) bool {
	rate, err := rates.At(time.Now())
	return err == nil && limit != nil && scale*rate.Value <= scale*(*limit)
}

// smartCostNextStart returns the next start time for a smart cost rate below the limit
func (lp *Loadpoint) smartCostNextStart(limit *float64, rates api.Rates, scale float64) time.Time {
	if limit == nil || rates == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rates {
		if slot.Start.After(now) && scale*slot.Value <= scale*(*limit) {
			return slot.Start
		}
	}

	return time.Time{}
}
