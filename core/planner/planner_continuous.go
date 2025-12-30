package planner

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

// findContinuousWindow finds the cheapest continuous window of slots for the given duration.
// - rates are filtered to [now, targetTime] window by caller
// Returns the selected rates.
func findContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) api.Rates {
	var bestCost *float64
	var bestIndex *int

	for i := range rates {
		windowEnd := rates[i].Start.Add(effectiveDuration)
		if windowEnd.After(targetTime) {
			break
		}

		cost := lo.SumBy(clampRates(rates[i:], rates[i].Start, windowEnd), func(r api.Rate) float64 {
			return float64(r.End.Sub(r.Start)) * r.Value
		})

		// TODO falls es das braucht fehlt ein Test

		// // only consider complete windows
		// if duration < effectiveDuration {
		// 	continue
		// }

		// Prefer later start if equal cost
		if bestCost == nil || cost <= *bestCost {
			bestCost = &cost
			bestIndex = &i
		}
	}

	// No valid window found
	if bestIndex == nil {
		return nil
	}

	// Build the best window only once
	windowEnd := rates[*bestIndex].Start.Add(effectiveDuration)

	return clampRates(rates[*bestIndex:], rates[*bestIndex].Start, windowEnd)
}
