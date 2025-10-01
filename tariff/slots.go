package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

type SlotWrapper struct {
	api.Tariff
}

func (t *SlotWrapper) Rates() (api.Rates, error) {
	rates, err := t.Tariff.Rates()
	if err != nil {
		return nil, err
	}
	return convertTo15mSlots(rates, t.Type()), nil
}

// convertTo15mSlots converts arbitrary slot lengths (e.g. 1h, 30m) to 15m slots.
// For price tariffs, the value is constant over all sub-slots.
// For solar/co2, linear interpolation is used between slot boundaries.
func convertTo15mSlots(rates api.Rates, typ api.TariffType) api.Rates {
	const slot = 15 * time.Minute
	var result api.Rates

	now := time.Now().Truncate(slot)

	for i, r := range rates {
		if !r.End.After(now) { // only keep slots >= now
			continue
		}

		interval := r.End.Sub(r.Start)
		numSlots := max(int(interval/slot), 1)

		for j := range numSlots {
			start := r.Start.Add(time.Duration(j) * slot)

			end := start.Add(slot)
			var val float64

			switch typ {
			case api.TariffTypePriceStatic, api.TariffTypePriceDynamic, api.TariffTypePriceForecast:
				val = r.Value

			case api.TariffTypeSolar, api.TariffTypeCo2:
				if i+1 < len(rates) {
					start0 := r.Start
					start1 := rates[i+1].Start
					frac := float64(start.Sub(start0)) / float64(start1.Sub(start0))
					val = r.Value + frac*(rates[i+1].Value-r.Value)
				} else {
					val = r.Value
				}

			default:
				panic("invalid tariff type: " + typ.String())
			}

			result = append(result, api.Rate{
				Start: start,
				End:   end,
				Value: val,
			})
		}
	}
	return result
}
