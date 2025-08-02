package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
)

func search(rr api.Rates, ts time.Time) (int, bool) {
	return slices.BinarySearchFunc(rr, ts, func(v api.Rate, ts time.Time) int {
		return v.Start.Compare(ts)
	})
}

// interpolate returns the interpolated value where ts is between two entries and i is the index of the rate after ts
func interpolate(rr api.Rates, i int, ts time.Time) float64 {
	rp := &rr[i-1]
	r := &rr[i]
	return rp.Value + float64(ts.Sub(rp.Start))*(r.Value-rp.Value)/float64(r.Start.Sub(rp.Start))
}

func value(rr api.Rates, ts time.Time) float64 {
	idx, ok := search(rr, ts)
	if ok {
		return rr[idx].Value
	}
	if idx == 0 || idx >= len(rr) {
		return 0
	}
	return interpolate(rr, idx, ts)
}

// energy calculates the energy consumption between from and to,
// assuming the rates containing the power at given timestamp.
// Result is in Wh
func energy(rr api.Rates, from, to time.Time) float64 {
	var energy float64

	idx, ok := search(rr, from)
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
			vp := interpolate(rr, idx, from)

			// to is before same entry as from
			if r.Start.After(to) {
				return (vp + interpolate(rr, idx, to)) / 2 * to.Sub(from).Hours()
			}

			energy += (vp + r.Value) / 2 * r.Start.Sub(from).Hours()
		}
	}

	for ; idx < len(rr)-1; idx++ {
		r := &rr[idx]
		rn := &rr[idx+1]

		if rn.Start.After(to) {
			energy += (r.Value + interpolate(rr, idx+1, to)) / 2 * to.Sub(r.Start).Hours()
			break
		}

		energy += (r.Value + rn.Value) / 2 * rn.Start.Sub(r.Start).Hours()
	}

	return energy
}
