package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

// timeseries is a sorted list of timestamped values
// methods are optimized for fast searching and interpolation
type timeseries []tsval

type tsval struct {
	Timestamp time.Time `json:"ts"`
	Value     float64   `json:"val"`
}

func (rr timeseries) search(ts time.Time) (int, bool) {
	return slices.BinarySearchFunc(rr, ts, func(v tsval, ts time.Time) int {
		return v.Timestamp.Compare(ts)
	})
}

// interpolate returns the interpolated value where ts is between two entries and i is the index of the rate after ts
func (rr timeseries) interpolate(i int, ts time.Time) float64 {
	rp := &rr[i-1]
	r := &rr[i]
	return rp.Value + float64(ts.Sub(rp.Timestamp))*(r.Value-rp.Value)/float64(r.Timestamp.Sub(rp.Timestamp))
}

func (rr timeseries) value(ts time.Time) float64 {
	idx, ok := rr.search(ts)
	if ok {
		return rr[idx].Value
	}
	if idx == 0 || idx >= len(rr) {
		return 0
	}
	return rr.interpolate(idx, ts)
}

// energy calculates the energy consumption between from and to,
// assuming the rates containing the power at given timestamp.
// Result is in Wh
func (rr timeseries) energy(from, to time.Time) float64 {
	var energy float64

	idx, ok := rr.search(from)
	if !ok {
		switch {
		case idx >= len(rr):
			// from is just before or after last entry
			return 0
		case idx == 0:
			// from is before first entry
			// do nothing- we ignore anything before the first entry
		default:
			// from is between two entries
			r := &rr[idx]
			vp := rr.interpolate(idx, from)

			// to is before same entry as from
			if r.Timestamp.After(to) {
				return (vp + rr.interpolate(idx, to)) / 2 * to.Sub(from).Hours()
			}

			energy += (vp + r.Value) / 2 * r.Timestamp.Sub(from).Hours()
		}
	}

	for ; idx < len(rr)-1; idx++ {
		r := &rr[idx]
		rn := &rr[idx+1]

		if rn.Timestamp.After(to) {
			energy += (r.Value + rr.interpolate(idx+1, to)) / 2 * to.Sub(r.Timestamp).Hours()
			break
		}

		energy += (r.Value + rn.Value) / 2 * rn.Timestamp.Sub(r.Timestamp).Hours()
	}

	return energy
}

func timestampSeries(rr api.Rates) timeseries {
	return lo.Map(rr, func(r api.Rate, _ int) tsval {
		return tsval{
			Timestamp: r.Start,
			Value:     r.Value,
		}
	})
}
