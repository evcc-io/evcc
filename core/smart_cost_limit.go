package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

type smartCostLimitPercentGetter interface {
	GetSmartCostLimitPercent() *float64
}

func effectiveSmartCostLimit(lp loadpoint.API, rates api.Rates) (*float64, bool) {
	percent := smartCostLimitPercent(lp)
	if percent == nil {
		return lp.GetSmartCostLimit(), false
	}

	average, ok := averageRateValue(rates)
	if !ok {
		return nil, true
	}

	limit := average * *percent / 100
	return &limit, true
}

func smartCostLimitPercent(lp loadpoint.API) *float64 {
	if getter, ok := lp.(smartCostLimitPercentGetter); ok {
		return getter.GetSmartCostLimitPercent()
	}
	return nil
}

func averageRateValue(rates api.Rates) (float64, bool) {
	if len(rates) == 0 {
		return 0, false
	}

	var sum float64
	var total time.Duration
	for _, rate := range rates {
		duration := rate.End.Sub(rate.Start)
		if duration <= 0 {
			continue
		}
		sum += rate.Value * duration.Seconds()
		total += duration
	}

	if total == 0 {
		return 0, false
	}

	return sum / total.Seconds(), true
}
