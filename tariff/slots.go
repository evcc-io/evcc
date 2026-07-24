package tariff

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
)

const SlotDuration = 15 * time.Minute

type SlotWrapper struct {
	api.Tariff
}

// Rates converts arbitrary slot lengths (e.g. 1h, 30m) to 15m slots.
// Slot length must be multiple of SlotDuration.
// For price tariffs, the value is constant over all sub-slots.
// For solar, the slot energy is distributed over the sub-slots.
func (t *SlotWrapper) Rates() (api.Rates, error) {
	rates, err := t.Tariff.Rates()
	if err != nil {
		return nil, err
	}

	if t.Type() == api.TariffTypeSolar {
		rates = aggregateEnergy(rates)
	}

	var res api.Rates
	if len(rates) > 0 {
		// assume all slots of equal length
		res = make(api.Rates, 0, len(rates)*max(int(rates[0].End.Sub(rates[0].Start)/SlotDuration), 1))
	}

	now := time.Now().Truncate(SlotDuration)

	for i, r := range rates {
		if !r.End.After(now) { // only keep slots >= now
			continue
		}

		numSlots := max(int(r.End.Sub(r.Start)/SlotDuration), 1)

		var vals []float64
		switch t.Type() {
		case api.TariffTypeSolar:
			vals = splitEnergy(rates, i, numSlots)
		default:
			vals = make([]float64, numSlots)
			for j := range vals {
				vals[j] = r.Value
			}
		}

		for j := range numSlots {
			start := r.Start.Add(time.Duration(j) * SlotDuration)

			res = append(res, api.Rate{
				Start: start,
				End:   start.Add(SlotDuration),
				Value: vals[j],
			})
		}
	}

	return res, nil
}

// aggregateEnergy merges sorted rates shorter than SlotDuration into aligned slots.
// Longer rates are left to splitEnergy which preserves the forecast curve.
func aggregateEnergy(rr api.Rates) api.Rates {
	if !slices.ContainsFunc(rr, func(r api.Rate) bool {
		d := r.End.Sub(r.Start)
		return d > 0 && d < SlotDuration
	}) {
		return rr
	}

	res := make(api.Rates, 0, len(rr))

	for _, r := range rr {
		d := r.End.Sub(r.Start)
		if d <= 0 {
			continue
		}

		for start := r.Start.Truncate(SlotDuration); start.Before(r.End); start = start.Add(SlotDuration) {
			from, to := start, start.Add(SlotDuration)
			if from.Before(r.Start) {
				from = r.Start
			}
			if to.After(r.End) {
				to = r.End
			}

			val := r.Value * float64(to.Sub(from)) / float64(d)

			if n := len(res); n > 0 && res[n-1].Start.Equal(start) {
				res[n-1].Value += val
				continue
			}

			res = append(res, api.Rate{Start: start, End: start.Add(SlotDuration), Value: val})
		}
	}

	return res
}

// splitEnergy distributes the energy of rate i over numSlots sub-slots, shaped by
// linear interpolation against the neighbouring slots' average power.
func splitEnergy(rates api.Rates, i, numSlots int) []float64 {
	r := rates[i]

	if numSlots <= 1 {
		return []float64{r.Value}
	}

	power := func(k int) float64 {
		d := rates[k].End.Sub(rates[k].Start).Hours()
		if d <= 0 {
			return 0
		}
		return rates[k].Value / d
	}

	cur := power(i)
	prev, next := cur, cur
	if i > 0 {
		prev = power(i - 1)
	}
	if i+1 < len(rates) {
		next = power(i + 1)
	}

	res := make([]float64, numSlots)

	var sum float64
	for j := range res {
		// sub-slot midpoint relative to the slot midpoint, [-0.5,0.5)
		f := (float64(j)+0.5)/float64(numSlots) - 0.5

		v := cur + f*(next-cur)*2
		if f < 0 {
			v = cur + f*(cur-prev)*2
		}

		res[j] = max(v, 0)
		sum += res[j]
	}

	for j := range res {
		if sum > 0 {
			res[j] = r.Value * res[j] / sum
		} else {
			res[j] = r.Value / float64(numSlots)
		}
	}

	return res
}
