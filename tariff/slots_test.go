package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTariff struct {
	rates api.Rates
	typ   api.TariffType
}

func (t *testTariff) Rates() (api.Rates, error) {
	return t.rates, nil
}
func (t *testTariff) Type() api.TariffType {
	return t.typ
}

// makeRates creates n consecutive rates starting at 'start', each with the given duration
// Values start at startVal and increase by 1 for each subsequent rate
func makeRates(start time.Time, duration time.Duration, n int, startVal float64) api.Rates {
	var rates api.Rates
	for i := range n {
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
		w := &SlotWrapper{&testTariff{
			rates: rates,
			typ:   api.TariffTypePriceStatic,
		}}

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
	r0 := api.Rate{
		Start: now,
		End:   now.Add(15 * time.Minute),
		Value: 1.0,
	}

	// second: an hour that follows immediately
	r1 := api.Rate{
		Start: r0.End,
		End:   r0.End.Add(1 * time.Hour),
		Value: 3.0,
	}

	w := &SlotWrapper{&testTariff{
		rates: api.Rates{r0, r1},
		typ:   api.TariffTypePriceStatic,
	}}

	out, err := w.Rates()
	require.NoError(t, err)

	// expected: one 15m slot with value 1.0, then four 15m slots with value 3.0
	expected := api.Rates{api.Rate{
		Start: r0.Start,
		End:   r0.End,
		Value: 1.0,
	}}

	for i := range 4 {
		expected = append(expected, api.Rate{
			Start: r1.Start.Add(time.Duration(i) * SlotDuration),
			End:   r1.Start.Add(time.Duration(i+1) * SlotDuration),
			Value: 3.0,
		})
	}

	assert.Equal(t, expected, out)
}

func TestDropOldRates(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	// old rate that should be removed by the wrapper (ends before 'now')
	old := api.Rate{
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Value: 0.5,
	}

	w := &SlotWrapper{&testTariff{
		rates: api.Rates{old},
		typ:   api.TariffTypeSolar,
	}}

	res, err := w.Rates()
	require.NoError(t, err)
	require.Len(t, res, 0)
}

// TestSolarEnergySplit verifies that splitting an hourly solar rate into 15min
// slots preserves the slot energy while following the neighbouring slots' shape
func TestSolarEnergySplit(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	// two consecutive hourly solar rates: no energy in the first hour, 4Wh in the next
	r0 := api.Rate{
		Start: now,
		End:   now.Add(time.Hour),
		Value: 0.0,
	}
	r1 := api.Rate{
		Start: r0.End,
		End:   r0.End.Add(time.Hour),
		Value: 4.0,
	}

	w := &SlotWrapper{&testTariff{
		rates: api.Rates{r0, r1},
		typ:   api.TariffTypeSolar,
	}}

	res, err := w.Rates()
	require.NoError(t, err)
	require.Len(t, res, 8)

	for i, r := range res {
		assert.Equal(t, now.Add(time.Duration(i)*SlotDuration), r.Start, "slot %d", i)
		assert.Equal(t, SlotDuration, r.End.Sub(r.Start), "slot %d", i)
	}

	// energy is conserved per source slot
	var first, second float64
	for i, r := range res {
		if i < 4 {
			first += r.Value
		} else {
			second += r.Value
		}
	}
	assert.InDelta(t, r0.Value, first, 1e-9)
	assert.InDelta(t, r1.Value, second, 1e-9)

	// ramping up from the empty hour, flat towards the missing successor
	for _, tc := range []struct {
		idx      int
		expected float64
	}{
		{4, 4.0 / 12},
		{5, 12.0 / 12},
		{6, 16.0 / 12},
		{7, 16.0 / 12},
	} {
		assert.InDelta(t, tc.expected, res[tc.idx].Value, 1e-9, "slot %d", tc.idx)
	}
}
