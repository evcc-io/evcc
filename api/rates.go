package api

import (
	"errors"
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
		if (rr.Start.Before(now) || rr.Start.Equal(now)) && rr.End.After(now) {
			return rr, nil
		}
	}

	return Rate{}, errors.New("no matching rate")
}
