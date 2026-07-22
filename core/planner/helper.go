package planner

import (
	"cmp"
	"iter"
	"slices"
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

// clampRates filters rates to the given time window and adjusts boundary slots
func clampRates(rates api.Rates, start, end time.Time) api.Rates {
	res := make(api.Rates, 0, len(rates))
	return slices.AppendSeq(res, clampRatesSeq(rates, start, end))
}

// clampRatesSeq returns an iterator for filtering rates to the given time window and adjusts boundary slots
func clampRatesSeq(rates api.Rates, start, end time.Time) iter.Seq[api.Rate] {
	return func(yield func(api.Rate) bool) {
		for _, r := range rates {
			// slot before continuous plan
			if !r.End.After(start) {
				continue
			}

			// slot after continuous plan
			if !r.Start.Before(end) {
				return
			}

			// calculate adjusted bounds
			adjustedStart := r.Start
			if adjustedStart.Before(start) {
				adjustedStart = start
			}

			adjustedEnd := r.End
			if adjustedEnd.After(end) {
				adjustedEnd = end
			}

			// skip if adjustment would create invalid slot
			if !adjustedEnd.After(adjustedStart) {
				continue
			}

			if !yield(api.Rate{
				Start: adjustedStart,
				End:   adjustedEnd,
				Value: r.Value,
			}) {
				return // Stop early if yield returns false
			}
		}
	}
}

// windowCost returns the cost of the given window. Durations are summed per price before
// multiplication, hence the result does not depend on how the window's slots are clamped.
func windowCost(rates api.Rates, start, end time.Time) float64 {
	type pricedDuration struct {
		value float64
		dur   time.Duration
	}

	var acc []pricedDuration
	for r := range clampRatesSeq(rates, start, end) {
		dur := r.End.Sub(r.Start)

		if idx := slices.IndexFunc(acc, func(p pricedDuration) bool { return p.value == r.Value }); idx >= 0 {
			acc[idx].dur += dur
			continue
		}

		acc = append(acc, pricedDuration{r.Value, dur})
	}

	slices.SortFunc(acc, func(i, j pricedDuration) int { return cmp.Compare(i.value, j.value) })

	var cost float64
	for _, p := range acc {
		cost += p.value * float64(p.dur)
	}

	return cost
}

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

		cost := windowCost(rates[i:], rates[i].Start, windowEnd)

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
