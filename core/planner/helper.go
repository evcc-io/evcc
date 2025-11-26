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
		// defensive: skip invalid slots (should not occur if filters work correctly)
		if slotDuration > 0 {
			duration += slotDuration
		}
	}
	return duration
}

// AverageCost returns the time-weighted average cost
func AverageCost(plan api.Rates) float64 {
	var cost float64
	var duration time.Duration
	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		// defensive: skip invalid slots (should not occur if filters work correctly)
		if slotDuration <= 0 {
			continue
		}
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

// isValidSlot returns true if slot has positive duration
func isValidSlot(slot api.Rate) bool {
	return slot.End.After(slot.Start)
}

// adjustSlotStart trims slot start to the given time if it starts before
func adjustSlotStart(slot *api.Rate, start time.Time) {
	if slot.Start.Before(start) {
		slot.Start = start
	}
}

// adjustSlotEnd trims slot end to the given time if it extends beyond
func adjustSlotEnd(slot *api.Rate, end time.Time) {
	if slot.End.After(end) {
		slot.End = end
	}
}

// trimSlot trims excess duration from a slot
// trimEnd: true = trim end (prefer early start), false = trim start (prefer late start)
func trimSlot(slot *api.Rate, excess time.Duration, trimEnd bool) {
	if excess < 0 {
		panic("trimSlot: excess must not be negative")
	}

	if trimEnd {
		slot.End = slot.End.Add(-excess)
	} else {
		slot.Start = slot.Start.Add(excess)
	}

	// Ensure slot remains valid - if trimming would create invalid slot, make it zero-duration
	if slot.End.Before(slot.Start) {
		slot.End = slot.Start
	}
}
