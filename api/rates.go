package api

import (
	"fmt"
	"slices"
	"time"
)

// Rate is a grid tariff rate
type Rate struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Price float64   `json:"price"`
}

// IsEmpty returns is the rate is the zero value
func (r Rate) IsEmpty() bool {
	return r.Start.IsZero() && r.End.IsZero() && r.Price == 0
}

// Rates is a slice of (future) tariff rates
type Rates []Rate

// Sort rates by start time
func (r Rates) Sort() {
	slices.SortStableFunc(r, func(i, j Rate) int {
		return i.Start.Compare(j.Start)
	})
}

// Current returns the rates current rate or error
func (r Rates) Current(now time.Time) (Rate, error) {
	for _, rr := range r {
		if !rr.Start.After(now) && rr.End.After(now) {
			return rr, nil
		}
	}

	if len(r) == 0 {
		return Rate{}, fmt.Errorf("no matching rate for: %s", now.Local().Format(time.RFC3339))
	}
	return Rate{}, fmt.Errorf("no matching rate for: %s, %d rates (%s to %s)",
		now.Local().Format(time.RFC3339), len(r), r[0].Start.Local().Format(time.RFC3339), r[len(r)-1].End.Local().Format(time.RFC3339))
}
