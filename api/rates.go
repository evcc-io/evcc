package api

import (
	"errors"
	"time"
)

// Current returns the rates current rate or error
func (r Rates) Current(now time.Time) (Rate, error) {
	for _, rr := range r {
		if (rr.Start.Before(now) || rr.Start.Equal(now)) && rr.End.After(now) {
			return rr, nil
		}
	}

	return Rate{}, errors.New("no matching rate")
}
