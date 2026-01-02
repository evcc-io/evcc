package planner

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// Start returns the earliest slot's start time
func Start(plan api.Rates) time.Time {
	var start time.Time
	for _, slot := range plan {
		if start.IsZero() || slot.Start.Before(start) {
			start = slot.Start
		}
	}
	return start
}

func End(plan api.Rates) time.Time {
	var end time.Time
	for _, slot := range plan {
		if end.IsZero() || slot.End.After(end) {
			end = slot.End
		}
	}
	return end
}

// Duration returns the sum of all slot's durations
func Duration(plan api.Rates) time.Duration {
	var duration time.Duration
	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		duration += slotDuration
	}
	return duration
}

// AverageCost returns the time-weighted average cost
func AverageCost(plan api.Rates) float64 {
	var cost float64
	var duration time.Duration
	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		duration += slotDuration
		cost += float64(slotDuration) * slot.Value
	}
	if duration == 0 {
		return 0
	}
	return cost / float64(duration)
}

// SlotAt returns the slot for the given time or an empty slot
func SlotAt(time time.Time, plan api.Rates) api.Rate {
	for _, slot := range plan {
		if !slot.Start.After(time) && slot.End.After(time) {
			return slot
		}
	}
	return api.Rate{}
}

// SlotHasSuccessor returns if the slot has an immediate successor.
// Does not require the plan to be sorted by start time.
func SlotHasSuccessor(r api.Rate, plan api.Rates) bool {
	for _, slot := range plan {
		if r.End.Equal(slot.Start) {
			return true
		}
	}
	return false
}

// IsFirst returns if the slot is the first slot in the plan.
// Does not require the plan to be sorted by start time.
func IsFirst(r api.Rate, plan api.Rates) bool {
	for _, slot := range plan {
		if r.Start.After(slot.Start) {
			return false
		}
	}
	return true
}

// clampBounds returns the overlap of [rStart,rEnd] with [start,end]
// ok is true if there is any overlap.
func clampBounds(rStart, rEnd, start, end time.Time) (time.Time, time.Time, bool) {
	if rStart.Before(start) {
		rStart = start
	}
	if rEnd.After(end) {
		rEnd = end
	}
	return rStart, rEnd, rEnd.After(rStart)
}

// clampRates filters rates to the given time window and adjusts boundary slots
func clampRates(rates api.Rates, start, end time.Time) api.Rates {
	res := make(api.Rates, 0, len(rates))
	for _, r := range rates {
		if s, e, ok := clampBounds(r.Start, r.End, start, end); ok {
			res = append(res, api.Rate{Start: s, End: e, Value: r.Value})
		}
	}
	return res
}

// findContinuousWindow finds the cheapest continuous window of the given duration
// that ends before targetTime. Prefers later windows when costs are equal.
// Returns nil if no valid window exists.
func findContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) api.Rates {
	var (
		cost, bestCost    float64
		j, bestStart      int
		covered           time.Duration
		wasValid, hasBest bool
	)

	for i := range rates {
		start := rates[i].Start
		end := start.Add(effectiveDuration)
		if end.After(targetTime) {
			break
		}

		// reset the sliding window if the left pointer overtakes the right pointer
		// or the previous window did not cover the full effective duration
		if j < i || !wasValid {
			j = i
			cost = 0
			covered = 0
		}

		// expand the window to the right until the effective duration is covered
		for j < len(rates) && covered < effectiveDuration {
			if s, e, ok := clampBounds(rates[j].Start, rates[j].End, start, end); ok {
				d := e.Sub(s)
				cost += float64(d) * rates[j].Value
				covered += d
			}
			j++
		}

		// only consider windows where the total covered duration equals the desired duration
		wasValid = covered == effectiveDuration
		if wasValid {
			if !hasBest || cost <= bestCost {
				bestCost = cost
				bestStart = i
				hasBest = true
			}

			// slide window forward by removing the contribution of the current start interval
			if s, e, ok := clampBounds(rates[i].Start, rates[i].End, start, end); ok {
				d := e.Sub(s)
				cost -= float64(d) * rates[i].Value
				covered -= d
			}
		}
	}
	if !hasBest {
		return nil
	}
	start := rates[bestStart].Start
	end := start.Add(effectiveDuration)
	return clampRates(rates[bestStart:], start, end)
}
