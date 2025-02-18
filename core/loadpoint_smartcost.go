package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

func (lp *Loadpoint) smartCostActive(rates api.Rates) bool {
	rate, err := rates.At(time.Now())
	limit := lp.GetSmartCostLimit()
	return err == nil && limit != nil && rate.Price <= *limit
}

// smartCostNextStart returns the next start time for a smart cost rate below the limit
func (lp *Loadpoint) smartCostNextStart(rates api.Rates) time.Time {
	limit := lp.GetSmartCostLimit()
	if limit == nil || rates == nil {
		return time.Time{}
	}

	now := time.Now()
	for _, slot := range rates {
		if slot.Start.After(now) && slot.Price <= *limit {
			return slot.Start
		}
	}

	return time.Time{}
}
