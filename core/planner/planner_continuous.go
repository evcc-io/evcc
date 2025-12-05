package planner

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
)

// findContinuousWindow finds the cheapest continuous window of slots for the given duration.
// - rates are filtered to [now, targetTime] window by caller
// Returns the selected rates.
func findContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) api.Rates {
	bestCost := math.MaxFloat64
	bestStartIndex := -1

	for i := range rates {
		windowEnd := rates[i].Start.Add(effectiveDuration)

		if windowEnd.After(targetTime) {
			break
		}

		// Calculate cost and duration for this window (without building the array)
		var cost float64
		var duration time.Duration

		for j := i; j < len(rates) && duration < effectiveDuration; j++ {
			slot := rates[j]

			// slot partially or completely within window?
			if slot.Start.Before(windowEnd) {
				// calculate trimmed end if necessary
				slotEnd := slot.End
				if slotEnd.After(windowEnd) {
					slotEnd = windowEnd
				}

				slotDur := slotEnd.Sub(slot.Start)
				duration += slotDur
				cost += float64(slotDur) * slot.Value
			}
		}

		// only consider complete windows
		if duration < effectiveDuration {
			continue
		}

		// Prefer later start if equal cost
		if cost <= bestCost {
			bestCost = cost
			bestStartIndex = i
		}
	}

	// No valid window found
	if bestStartIndex == -1 {
		return nil
	}

	// Build the best window only once
	windowEnd := rates[bestStartIndex].Start.Add(effectiveDuration)
	var window api.Rates
	var duration time.Duration

	for j := bestStartIndex; j < len(rates) && duration < effectiveDuration; j++ {
		slot := rates[j]

		if slot.Start.Before(windowEnd) {
			// trim end if necessary
			if slot.End.After(windowEnd) {
				slot.End = windowEnd
			}

			window = append(window, slot)
			duration += slot.End.Sub(slot.Start)
		}
	}

	return window
}
