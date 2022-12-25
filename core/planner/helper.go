package planner

import (
	"time"

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

func Cost(plan api.Rates) float64 {
	var cost float64
	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		cost += float64(slotDuration) / float64(time.Hour) * slot.Price
	}
	return cost
}
