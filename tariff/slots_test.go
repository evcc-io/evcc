package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type priceTariff struct{ rates api.Rates }

func (t *priceTariff) Rates() (api.Rates, error) { return t.rates, nil }
func (t *priceTariff) Type() api.TariffType      { return api.TariffTypePriceStatic }

type solarTariff struct{ rates api.Rates }

func (t *solarTariff) Rates() (api.Rates, error) { return t.rates, nil }
func (t *solarTariff) Type() api.TariffType      { return api.TariffTypeSolar }

// makeRates creates n consecutive rates starting at 'start', each with the given duration
// Values start at startVal and increase by 1 for each subsequent rate
func makeRates(start time.Time, duration time.Duration, n int, startVal float64) api.Rates {
	var rates api.Rates
	for i := 0; i < n; i++ {
		s := start.Add(time.Duration(i) * duration)
		rates = append(rates, api.Rate{
			Start: s,
			End:   s.Add(duration),
			Value: startVal + float64(i),
		})
	}
	return rates
}

// TestBasicSlotConversionCounts ensures that different source durations are split into the expected number of 15-minute slots
func TestBasicSlotConversionCounts(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	cases := []struct {
		dur      time.Duration
		expected int
	}{
		{15 * time.Minute, 1},
		{30 * time.Minute, 2},
		{1 * time.Hour, 4},
		{2 * time.Hour, 8},
	}

	for _, tc := range cases {
		// Create a single rate of length tc.dur starting at "now"
		rates := makeRates(now, tc.dur, 1, 5.0)
		w := &SlotWrapper{&priceTariff{rates: rates}}

		out, err := w.Rates()
		require.NoError(t, err)

		// Check the number of produced 15-minute slots
		assert.Len(t, out, tc.expected, "duration %v", tc.dur)

		// Additional lightweight checks:
		// - first slot should begin at the original rate start
		// - every produced slot must have the configured SlotDuration length
		if len(out) > 0 {
			assert.Equal(t, rates[0].Start, out[0].Start)
			for _, r := range out {
				assert.Equal(t, SlotDuration, r.End.Sub(r.Start))
			}
		}
	}
}

// TestMixedSlots verifies a mix of a 15-minute rate followed by a 1-hour rate
// For price tariffs subslots from the hour should keep the same constant price
func TestMixedSlots(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	// first: a single 15-minute rate
	first := api.Rate{
		Start: now,
		End:   now.Add(15 * time.Minute),
		Value: 1.0,
	}
	// second: an hour that follows immediately
	second := api.Rate{
		Start: first.End,
		End:   first.End.Add(1 * time.Hour),
		Value: 3.0,
	}

	w := &SlotWrapper{&priceTariff{rates: api.Rates{first, second}}}
	out, err := w.Rates()
	require.NoError(t, err)

	// expected: one 15m slot with value 1.0, then four 15m slots with value 3.0
	var expected api.Rates
	expected = append(expected, api.Rate{Start: first.Start, End: first.End, Value: 1.0})

	s := second.Start
	for i := 0; i < 4; i++ {
		expected = append(expected, api.Rate{
			Start: s.Add(time.Duration(i) * SlotDuration),
			End:   s.Add(time.Duration(i+1) * SlotDuration),
			Value: 3.0,
		})
	}

	assert.Equal(t, expected, out)
}

// TestDropOldRatesAndSolarInterpolation checks two behaviors:
// 1) Old rates (that already ended) are dropped
// 2) Solar tariffs are linearly interpolated between consecutive rate boundaries
func TestDropOldRatesAndSolarInterpolation(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	// old rate that should be removed by the wrapper (ends before 'now')
	old := api.Rate{
		Start: now.Add(-2 * time.Hour),
		End:   now.Add(-1 * time.Hour),
		Value: 0.5,
	}

	// Two consecutive hourly solar rates: 0.0 in the first hour, 4.0 in the next
	// With linear interpolation, the first hour's four 15m slots should have values 0,1,2,3
	r0 := api.Rate{
		Start: now,
		End:   now.Add(1 * time.Hour),
		Value: 0.0,
	}
	r1 := api.Rate{
		Start: r0.End,
		End:   r0.End.Add(1 * time.Hour),
		Value: 4.0,
	}

	w := &SlotWrapper{&solarTariff{rates: api.Rates{old, r0, r1}}}
	out, err := w.Rates()
	require.NoError(t, err)

	// Build expected results: skip the old rate, then r0 interpolated into 4 slots (0..3), then r1 as four slots with value 4.0
	var expected api.Rates
	for j := 0; j < 4; j++ {
		expected = append(expected, api.Rate{
			Start: r0.Start.Add(time.Duration(j) * SlotDuration),
			End:   r0.Start.Add(time.Duration(j+1) * SlotDuration),
			Value: float64(j),
		})
	}
	for j := 0; j < 4; j++ {
		expected = append(expected, api.Rate{
			Start: r1.Start.Add(time.Duration(j) * SlotDuration),
			End:   r1.Start.Add(time.Duration(j+1) * SlotDuration),
			Value: 4.0,
		})
	}

	assert.Equal(t, expected, out)
}
