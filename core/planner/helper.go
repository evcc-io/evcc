package planner

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
)

func Start(plan api.Rates) time.Time {
	var start time.Time
	for _, slot := range plan {
		if start.IsZero() || slot.Start.Before(start) {
			start = slot.Start
		}
	}
	return start
}

func Duration(plan api.Rates) time.Duration {
	var duration time.Duration
	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		duration += slotDuration
	}
	return duration
}

func AverageCost(plan api.Rates) float64 {
	var cost float64
	var duration time.Duration
	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		duration += slotDuration
		cost += float64(slotDuration) * slot.Price
	}
	return cost / float64(duration)
}

func ActiveSlot(clock clock.Clock, plan api.Rates) api.Rate {
	for _, slot := range plan {
		if (slot.Start.Before(clock.Now()) || slot.Start.Equal(clock.Now())) && slot.End.After(clock.Now()) {
			return slot
		}
	}
	return api.Rate{}
}
