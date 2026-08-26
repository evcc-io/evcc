package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

const SlotDuration = 15 * time.Minute

type SlotWrapper struct {
	api.Tariff
}

// Rates converts arbitrary slot lengths (multiples of SlotDuration) to 15m slots.
// Price sub-slots are constant, solar sub-slots interpolated towards the next slot.
func (t *SlotWrapper) Rates() (api.Rates, error) {
	rates, err := t.Tariff.Rates()
	if err != nil {
		return nil, err
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

		vals := make([]float64, numSlots)
		for j := range vals {
			vals[j] = r.Value
		}

		if t.Type() == api.TariffTypeSolar {
			shapeSolar(rates, i, vals)
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

// shapeSolar samples the source curve at the sub-slot starts. A solar value is the
// power at its Start (see core.solarEnergy), so splitting a slot must interpolate
// towards the successor rather than redistribute the value across the sub-slots -
// only then does the split leave the integrated energy untouched. Interpolation runs
// over the distance between the two starts, which is the slot length only for a gapless
// series. The trailing slot has no successor and stays flat.
func shapeSolar(rates api.Rates, i int, vals []float64) {
	cur := rates[i].Value

	for j := range vals {
		vals[j] = cur
	}

	if i+1 >= len(rates) {
		return
	}

	next := rates[i+1]
	span := next.Start.Sub(rates[i].Start)
	if span <= 0 {
		return
	}

	for j := range vals {
		// beyond the successor's start the curve is the successor's business
		d := min(time.Duration(j)*SlotDuration, span)
		vals[j] = cur + (next.Value-cur)*float64(d)/float64(span)
	}
}
