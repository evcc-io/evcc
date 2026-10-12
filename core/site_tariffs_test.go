package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
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

func TestLoadpointShares(t *testing.T) {
	tc := []struct {
		name                    string
		pv, battery, from, to   float64
		threshold, soc          float64
		capacity                float64
		greenShare, feedInShare float64
	}{
		{"pv only", 1000, 0, 0, 1000, 50, 30, 10, 1, 1},
		{"battery charging", 2000, -1000, 0, 1000, 50, 30, 10, 1, 1},
		{"zero power nothing available", 0, 0, 0, 0, 50, 30, 10, 0, 0},
		{"threshold off", 0, 1000, 0, 1000, 0, 30, 10, 1, 1},
		{"soc above threshold", 0, 1000, 0, 1000, 50, 80, 10, 1, 1},
		{"soc at threshold", 0, 1000, 0, 1000, 50, 50, 10, 1, 1},
		{"soc below threshold", 0, 1000, 0, 1000, 50, 30, 10, 1, 0},
		{"soc never read", 0, 1000, 0, 1000, 50, 0, 0, 1, 1},
		{"pv covers lower consumption, battery the rest", 500, 1000, 500, 1500, 50, 30, 10, 1, 0},
		{"pv and battery split window", 1000, 1000, 500, 2500, 50, 30, 10, 0.75, 0.25},
		{"zero power with battery available", 0, 1000, 0, 0, 50, 30, 10, 1, 0},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			var state siteState
			state.pvPower = tc.pv
			state.battery.Power = tc.battery
			state.battery.Soc = tc.soc
			state.battery.Capacity = tc.capacity

			green, feedIn := loadpointShares(state, tc.threshold, tc.from, tc.to)
			assert.InDelta(t, tc.greenShare, green, 1e-9, "green share")
			assert.InDelta(t, tc.feedInShare, feedIn, 1e-9, "feed-in share")
			assert.InDelta(t, tc.greenShare, greenShare(state, tc.from, tc.to), 1e-9, "green share unaffected by threshold")
		})
	}
}

func TestEffectivePrice(t *testing.T) {
	grid, err := tariff.NewFixedFromConfig(map[string]any{"price": 0.4})
	require.NoError(t, err)
	feedin, err := tariff.NewFixedFromConfig(map[string]any{"price": 0.1})
	require.NoError(t, err)

	tc := []struct {
		name        string
		noFeedIn    bool
		feedInShare float64
		price       float64
	}{
		{"all feed-in", false, 1, 0.1},
		{"all grid", false, 0, 0.4},
		{"mixed", false, 0.25, 0.4*0.75 + 0.1*0.25},
		{"no feed-in tariff", true, 1, 0},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			site := &Site{tariffs: &tariff.Tariffs{Grid: grid, FeedIn: feedin}}
			if tc.noFeedIn {
				site.tariffs.FeedIn = nil
			}

			price := site.effectivePrice(tc.feedInShare)
			require.NotNil(t, price)
			assert.InDelta(t, tc.price, *price, 1e-9)
		})
	}
}
