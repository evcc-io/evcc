package core

import (
	"encoding/json"
	"time"

	"github.com/evcc-io/evcc/api"
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

// solarPower converts the energy of a solar rate into average power in W
func solarPower(r api.Rate) float64 {
	d := r.End.Sub(r.Start).Hours()
	if d <= 0 {
		return 0
	}
	return r.Value / d
}

// solarTimeseries converts solar energy rates into power at slot start. The average
// power of a slot is centered, so the boundary power is interpolated from both slots.
func solarTimeseries(rr api.Rates) []tsEntry {
	res := make([]tsEntry, 0, len(rr))

	for i, r := range rr {
		power := solarPower(r)
		if i > 0 {
			power = (solarPower(rr[i-1]) + power) / 2
		}

		res = append(res, tsEntry{Timestamp: r.Start, Value: power})
	}

	return res
}

// solarEnergy sums the energy of all rates overlapping [from,to), prorating partial slots.
// Result is in Wh
func solarEnergy(rr api.Rates, from, to time.Time) float64 {
	if from.After(to) {
		panic("from cannot be after to")
	}

	var energy float64

	for _, r := range rr {
		d := r.End.Sub(r.Start)
		if d <= 0 {
			continue
		}

		start, end := r.Start, r.End
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		if !end.After(start) {
			continue
		}

		energy += r.Value * float64(end.Sub(start)) / float64(d)
	}

	return energy
}
