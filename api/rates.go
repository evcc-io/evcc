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

// implement sort.Interface
func (r Rates) Len() int {
	return len(r)
}

func (r Rates) Less(i, j int) bool {
	if r[i].Price == r[j].Price {
		return r[i].Start.After(r[j].Start)
	}

	return r[i].Price < r[j].Price
}

func (r Rates) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
