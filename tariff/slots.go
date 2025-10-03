package tariff

import (
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
// For solar/co2, linear interpolation is used between slot boundaries.
func (t *SlotWrapper) Rates() (api.Rates, error) {
	rates, err := t.Tariff.Rates()
	if err != nil {
		return nil, err
	}

	var res api.Rates

	now := time.Now().Truncate(SlotDuration)

	for i, r := range rates {
		if !r.End.After(now) { // only keep slots >= now
			continue
		}

		interval := r.End.Sub(r.Start)
		numSlots := max(int(interval/SlotDuration), 1)

		for j := range numSlots {
			start := r.Start.Add(time.Duration(j) * SlotDuration)
			end := start.Add(SlotDuration)

			var val float64

			switch t.Type() {
			case api.TariffTypeSolar: //, api.TariffTypeCo2
				if i+1 < len(rates) {
					start0 := r.Start
					start1 := rates[i+1].Start
					frac := float64(start.Sub(start0)) / float64(start1.Sub(start0))
					val = r.Value + frac*(rates[i+1].Value-r.Value)
				} else {
					val = r.Value
				}

			default:
				val = r.Value
			}

			res = append(res, api.Rate{
				Start: start,
				End:   end,
				Value: val,
			})
		}
	}

	return res, nil
}
