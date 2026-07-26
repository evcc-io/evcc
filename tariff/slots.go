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
// Price sub-slots are constant, solar sub-slots interpolated around the slot center.
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

// shapeSolar interpolates solar sub-slots between the slot centers. The slot value
// applies to the entire period, so sub-slots are centered and rescaled to it. Slot
// edges meet the average of both neighbouring values, keeping the curve continuous.
func shapeSolar(rates api.Rates, i int, vals []float64) {
	if len(vals) < 2 {
		return
	}

	cur := rates[i].Value
	if cur <= 0 {
		// empty slot stays empty, shaping a non-positive slot would flip signs when rescaling
		return
	}

	prev, next := cur, cur
	if i > 0 {
		prev = rates[i-1].Value
	}
	if i+1 < len(rates) {
		next = rates[i+1].Value
	}

	var sum float64
	for j := range vals {
		// sub-slot midpoint relative to the slot midpoint, [-0.5,0.5)
		f := (float64(j)+0.5)/float64(len(vals)) - 0.5

		delta := next - cur
		if f < 0 {
			delta = cur - prev
		}

		vals[j] = max(cur+f*delta, 0)
		sum += vals[j]
	}

	if sum <= 0 {
		for j := range vals {
			vals[j] = cur
		}
		return
	}

	// preserve the slot average
	scale := cur * float64(len(vals)) / sum
	for j := range vals {
		vals[j] *= scale
	}
}
