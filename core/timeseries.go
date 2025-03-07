package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

type (
	timeseries []tsval
	tsval      struct {
		Timestamp time.Time `json:"ts"`
		Value     float64   `json:"val"`
	}
)

func (rr timeseries) index(ts time.Time) (int, bool) {
	return slices.BinarySearchFunc(rr, ts, func(v tsval, ts time.Time) int {
		return v.Timestamp.Compare(ts)
	})
}

func (rr timeseries) interpolate(i int, ts time.Time) float64 {
	return float64(ts.Sub(rr[i-1].Timestamp)) * (rr[i].Value - rr[i-1].Value) / float64(rr[i].Timestamp.Sub(rr[i-1].Timestamp))
}

func (rr timeseries) value(ts time.Time) float64 {
	index, ok := rr.index(ts)
	if ok {
		return rr[index].Value
	}
	if index == 0 || index >= len(rr) {
		return 0
	}
	return rr.interpolate(index, ts)
}

// energy calculates the energy consumption between from and to,
// assuming the rates containing the power at given timestamp.
// Result is in Wh
func (rr timeseries) energy(from, to time.Time) float64 {
	var energy float64

	idx, ok := rr.index(from)
	if !ok {
		if idx == 0 || idx >= len(rr) {
			return 0
		}
		idx--
	}

	last := rr[idx]

	// for _, r := range rr[idx+1:] {
	for idx++; idx < len(rr); idx++ {
		r := rr[idx]
		// fmt.Println(r.Start.Local().Format(time.RFC3339), r.End.Local().Format(time.RFC3339), r.Price)

		x1 := last.Timestamp
		y1 := last.Value
		if x1.Before(from) {
			x1 = from
			y1 += float64(from.Sub(last.Timestamp)) * (r.Value - last.Value) / float64(r.Timestamp.Sub(last.Timestamp))
		}

		x2 := r.Timestamp
		y2 := r.Value
		if x2.After(to) {
			x2 = to
			y2 += float64(to.Sub(r.Timestamp)) * (r.Value - last.Value) / float64(r.Timestamp.Sub(last.Timestamp))
		}

		energy += (y1 + y2) / 2 * x2.Sub(x1).Hours()

		if !r.Timestamp.Before(to) {
			break
		}

		last = r
	}

	return energy
}

func timestampSeries(rr api.Rates) timeseries {
	return lo.Map(rr, func(r api.Rate, _ int) tsval {
		return tsval{
			Timestamp: r.Start,
			Value:     r.Price,
		}
	})
}
