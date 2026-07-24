package core

import (
	"encoding/json"
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

// solarPower converts the energy of a solar rate into average power in W
func solarPower(r api.Rate) float64 {
	d := r.End.Sub(r.Start).Hours()
	if d <= 0 {
		return 0
	}
	return r.Value / d
}

// solarTimeseries converts solar energy rates into power at timestamp
func solarTimeseries(rr api.Rates) []tsEntry {
	return lo.Map(rr, func(r api.Rate, _ int) tsEntry {
		return tsEntry{Timestamp: r.Start, Value: solarPower(r)}
	})
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
