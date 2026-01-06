package core

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

type timeseries []tsEntry

var _ api.BytesMarshaler = (*timeseries)(nil)

func (ts timeseries) MarshalBytes() ([]byte, error) {
	return json.Marshal(ts)
}

type tsEntry struct {
	Timestamp time.Time `json:"ts"`
	Value     float64   `json:"val"`
}

func solarTimeseries(rr api.Rates) []tsEntry {
	return lo.Map(rr, func(r api.Rate, _ int) tsEntry {
		return tsEntry{Timestamp: r.Start, Value: r.Value}
	})
}

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

// solarEnergy calculates the energy consumption between from and to,
// assuming the rates containing the power at given timestamp.
// Result is in Wh
func solarEnergy(rr api.Rates, from, to time.Time) float64 {
	var energy float64

	if from.After(to) {
		panic("from cannot be after to")
	}

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
