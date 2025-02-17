package api

import (
	"encoding/json"
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
func (rr Rates) Sort() {
	slices.SortStableFunc(rr, func(i, j Rate) int {
		return i.Start.Compare(j.Start)
	})
}

// At returns the rate for given timestamp or error.
// Rates MUST be sorted by start time.
func (rr Rates) At(ts time.Time) (Rate, error) {
	if i, ok := slices.BinarySearchFunc(rr, ts, func(r Rate, t time.Time) int {
		switch {
		case ts.Before(r.Start):
			return +1
		case !ts.Before(r.End):
			return -1
		default:
			return 0
		}
	}); ok {
		return rr[i], nil
	}

	var zero Rate
	if len(rr) == 0 {
		return zero, fmt.Errorf("no matching rate for: %s", ts.Local().Format(time.RFC3339))
	}
	return zero, fmt.Errorf("no matching rate for: %s, %d rates (%s to %s)",
		ts.Local().Format(time.RFC3339), len(rr), rr[0].Start.Local().Format(time.RFC3339), rr[len(rr)-1].End.Local().Format(time.RFC3339))
}

// MarshalMQTT implements server.MQTTMarshaler
func (r Rates) MarshalMQTT() ([]byte, error) {
	return json.Marshal(r)
}
