package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestPlannerExtendsGrid(t *testing.T) {
	ctrl := gomock.NewController(t)

	start := time.Now().Truncate(tariff.SlotDuration)
	rates := func(from, to int, value float64) api.Rates {
		var res api.Rates
		for i := from; i < to; i++ {
			ts := start.Add(time.Duration(i) * tariff.SlotDuration)
			res = append(res, api.Rate{Start: ts, End: ts.Add(tariff.SlotDuration), Value: value})
		}
		return res
	}

	grid := api.NewMockTariff(ctrl)
	grid.EXPECT().Type().Return(api.TariffTypePriceForecast).AnyTimes()
	grid.EXPECT().Rates().Return(rates(0, 8, 0.30), nil).AnyTimes()

	planner := api.NewMockTariff(ctrl)
	planner.EXPECT().Type().Return(api.TariffTypePriceForecast).AnyTimes()
	planner.EXPECT().Rates().Return(rates(0, 20, 0.305), nil).AnyTimes()

	site := &Site{
		log:     util.NewLogger("foo"),
		tariffs: &tariff.Tariffs{Grid: grid, Planner: planner},
	}

	// off: grid only
	assert.Equal(t, grid, site.GetTariff(api.TariffUsageGrid))

	// on, but experimental disabled: grid only
	site.plannerExtendsGrid = true
	settings.SetBool(keys.Experimental, false)
	assert.Equal(t, grid, site.GetTariff(api.TariffUsageGrid))

	// on: grid extended by planner, planner itself unchanged
	settings.SetBool(keys.Experimental, true)
	t.Cleanup(func() { settings.SetBool(keys.Experimental, false) })
	res, err := site.GetTariff(api.TariffUsageGrid).Rates()
	require.NoError(t, err)
	require.Len(t, res, 20)
	assert.Equal(t, 0.30, res[7].Value, "grid price kept where known")
	assert.Equal(t, 0.305, res[8].Value, "planner price appended")
	assert.Equal(t, planner, site.GetTariff(api.TariffUsagePlanner))

	// co2 planner is not used to extend the grid price
	co2 := api.NewMockTariff(ctrl)
	co2.EXPECT().Type().Return(api.TariffTypeCo2).AnyTimes()
	site.tariffs.Planner = co2
	assert.Equal(t, grid, site.GetTariff(api.TariffUsageGrid))

	// no planner
	site.tariffs.Planner = nil
	assert.Equal(t, grid, site.GetTariff(api.TariffUsageGrid))
}
