package api

import (
	"encoding/json"
	"slices"
	"time"
)

// Rate is a grid tariff rate
type Rate struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Value float64   `json:"value"`
}

// IsZero returns is the rate is the zero value
func (r Rate) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero() && r.Value == 0
}

// Rates is a slice of (future) tariff rates
type Rates []Rate

// Sort rates by start time
func (rr Rates) Sort() {
	slices.SortStableFunc(rr, func(i, j Rate) int {
		return i.Start.Compare(j.Start)
	})
}

// At returns the rate for given timestamp or error.
// Rates MUST be sorted by start time.
func (rr Rates) At(ts time.Time) (Rate, error) {
	if i, ok := slices.BinarySearchFunc(rr, ts, func(r Rate, ts time.Time) int {
		switch {
		case ts.Before(r.Start):
			return +1
		case !ts.Before(r.End):
			return -1
		default:
			return 0
		}
	}); ok {
		return rr[i], nil
	}

	return Rate{}, ErrNotAvailable
}

// MarshalMQTT implements server.MQTTMarshaler
func (r Rates) MarshalMQTT() ([]byte, error) {
	return json.Marshal(r)
}

type Tariff15mWrapper struct {
	Inner Tariff
}

func (t Tariff15mWrapper) Rates() (Rates, error) {
	rates, err := t.Inner.Rates()
	if err != nil {
		return nil, err
	}
	return ConvertTo15mSlots(rates, t.Type()), nil
}

func (t Tariff15mWrapper) Type() TariffType {
	return t.Inner.Type()
}

// ConvertTo15mSlots converts arbitrary slot lengths (e.g. 1h, 30m) to 15m slots.
// For price tariffs, the value is constant over all sub-slots.
// For solar/co2, linear interpolation is used between slot boundaries.
func ConvertTo15mSlots(rates Rates, typ TariffType) Rates {
	const slot = 15 * time.Minute
	var result Rates

	now := time.Now().Truncate(slot)

	for i, r := range rates {
		interval := r.End.Sub(r.Start)
		numSlots := max(int(interval/slot), 1)
		for j := range numSlots {
			start := r.Start.Add(time.Duration(j) * slot)

			if start.Before(now) { // only keep slots >= now
				continue
			}

			end := start.Add(slot)
			var val float64

			switch typ {
			case TariffTypePriceStatic, TariffTypePriceDynamic, TariffTypePriceForecast:
				val = r.Value
			case TariffTypeSolar, TariffTypeCo2:
				if i+1 < len(rates) {
					start0 := r.Start
					start1 := rates[i+1].Start
					frac := float64(start.Sub(start0)) / float64(start1.Sub(start0))
					val = r.Value + frac*(rates[i+1].Value-r.Value)
				} else {
					val = r.Value
				}
			}
			result = append(result, Rate{Start: start, End: end, Value: val})
		}
	}
	return result
}
