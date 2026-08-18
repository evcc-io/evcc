package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForecastSlotEnergy(t *testing.T) {
	slot := time.Unix(1735689600, 0).Truncate(tariff.SlotDuration)
	rate := func(i int, power float64) api.Rate {
		return api.Rate{
			Start: slot.Add(time.Duration(i) * tariff.SlotDuration),
			End:   slot.Add(time.Duration(i+1) * tariff.SlotDuration),
			Value: power,
		}
	}

	rr := api.Rates{rate(0, 4000), rate(1, 8000)}

	// trapezoidal like the published curve, (4kW + 8kW) / 2 * 15min,
	// and identical anywhere inside the slot
	assert.Equal(t, 1.5, forecastSlotEnergy(rr, slot))
	assert.Equal(t, 1.5, forecastSlotEnergy(rr, slot.Add(time.Minute)))

	// the last sample has no successor to integrate towards, as for the daily totals
	assert.Equal(t, 0.0, forecastSlotEnergy(rr, slot.Add(tariff.SlotDuration)))

	// beyond the forecast horizon
	assert.Equal(t, 0.0, forecastSlotEnergy(rr, slot.Add(2*tariff.SlotDuration)))
}

func TestForecastRates(t *testing.T) {
	start := time.Unix(1735689600, 0)

	for _, tc := range []struct {
		desc  string
		rates api.Rates
		want  string
	}{
		{desc: "nil", rates: nil, want: "null"},
		{desc: "empty", rates: api.Rates{}, want: "null"},
		{
			desc: "slots",
			rates: api.Rates{
				{Start: start, End: start.Add(time.Hour), Value: 0.25},
				{Start: start.Add(time.Hour), End: start.Add(2 * time.Hour), Value: -0.1},
			},
			want: "[[1735689600,1735693200,0.25],[1735693200,1735696800,-0.1]]",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			b, err := json.Marshal(forecastRates(tc.rates))
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(b))
		})
	}
}

func TestTimeseriesMarshal(t *testing.T) {
	start := time.Unix(1735689600, 0)

	for _, tc := range []struct {
		desc string
		ts   timeseries
		want string
	}{
		{desc: "nil", ts: nil, want: "null"},
		{desc: "empty", ts: timeseries{}, want: "[]"},
		{
			desc: "entries",
			ts: timeseries{
				{Timestamp: start, Value: 1000},
				{Timestamp: start.Add(time.Hour), Value: 0},
			},
			want: "[[1735689600,1000],[1735693200,0]]",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			b, err := json.Marshal(tc.ts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(b))

			b, err = tc.ts.MarshalBytes()
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(b))
		})
	}
}

// TestQueryConsumptionSignal checks a fresh install with no consumption history yet:
// the signal should come back unscaled.
func TestQueryConsumptionSignal(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, metrics.SetupSchema())

	site := NewSite()
	col, err := metrics.NewCollector(metrics.Home, metrics.Home, metrics.Home)
	require.NoError(t, err)
	site.collectors[metrics.Home] = col

	sig, err := site.queryConsumptionSignal()
	require.NoError(t, err)
	assert.Equal(t, 1.0, sig)
}

func TestConsumptionSignalDecay(t *testing.T) {
	t.Run("no deviation gives the unscaled forecast", func(t *testing.T) {
		assert.InDelta(t, 1, consumptionSignalDecay(1, 0), 1e-9)
		assert.InDelta(t, 1, consumptionSignalDecay(1, 191), 1e-9)
	})

	t.Run("first slot returns the input signal unchanged, above and below baseline", func(t *testing.T) {
		for _, sig := range []float64{consumptionSignalMin, 0.7, 1.2, consumptionSignalMax} {
			assert.InDelta(t, sig, consumptionSignalDecay(sig, 0), 1e-9)
		}
	})

	t.Run("after 24h, exactly consumptionSignalPhiDay of the deviation remains", func(t *testing.T) {
		sig := 1.4
		want := 1 + consumptionSignalPhiDay*(sig-1)
		assert.InDelta(t, want, consumptionSignalDecay(sig, 24*4), 1e-9) // 96 slots = 24h
	})

	t.Run("deviation decays monotonically towards 1 over the horizon, above and below baseline", func(t *testing.T) {
		for _, sig := range []float64{1.3, 0.7} {
			prev := consumptionSignalDecay(sig, 0)
			for i := 1; i <= 192; i++ { // 48h at 15min slots
				cur := consumptionSignalDecay(sig, i)
				if sig > 1 {
					assert.LessOrEqual(t, cur, prev)
				} else {
					assert.GreaterOrEqual(t, cur, prev)
				}
				prev = cur
			}
			// never overshoots past 1
			if sig > 1 {
				assert.GreaterOrEqual(t, prev, 1.0)
			} else {
				assert.LessOrEqual(t, prev, 1.0)
			}
		}
	})

	t.Run("decay approaches 1 far into the horizon", func(t *testing.T) {
		// geometric decay, never fully zero: 0.41^5 ~= 0.0116 of the original deviation left after 5 days
		assert.InDelta(t, 1, consumptionSignalDecay(1.5, 480), 0.01)
	})
}

func TestPercentileOf(t *testing.T) {
	// n values of v
	fill := func(n int, v float64) []float64 {
		s := make([]float64, n)
		for i := range s {
			s[i] = v
		}
		return s
	}

	t.Run("too few samples returns false", func(t *testing.T) {
		_, ok := percentileOf(nil, 0.5, solarScaleMinSamples)
		assert.False(t, ok)

		_, ok = percentileOf(fill(solarScaleMinSamples-1, 0.9), 0.5, solarScaleMinSamples)
		assert.False(t, ok)
	})

	t.Run("stable cluster", func(t *testing.T) {
		v, ok := percentileOf(fill(20, 0.9), 0.5, solarScaleMinSamples)
		assert.True(t, ok)
		assert.InDelta(t, 0.9, v, 0.001)
	})

	// P50 rejects outlier days for free: a broken forecast feed (recent ratio
	// ~2.3) and a metering outage (ratio ~0.16) do not move the result as
	// long as they stay a minority of the window.
	t.Run("outlier days do not move P50", func(t *testing.T) {
		ratios := fill(20, 0.9)                   // healthy installation bias
		ratios = append(ratios, fill(4, 2.3)...)  // broken forecast feed
		ratios = append(ratios, fill(8, 0.16)...) // metering outage

		v, ok := percentileOf(ratios, 0.5, solarScaleMinSamples)
		assert.True(t, ok)
		assert.InDelta(t, 0.9, v, 0.001)
	})

	t.Run("higher percentile shifts toward the upper tail", func(t *testing.T) {
		ratios := append(fill(15, 0.8), fill(15, 1.2)...)

		p50, _ := percentileOf(ratios, 0.5, solarScaleMinSamples)
		p90, _ := percentileOf(ratios, 0.9, solarScaleMinSamples)
		assert.Less(t, p50, p90)
	})
}
