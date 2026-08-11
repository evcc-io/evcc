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
	slot := time.Now().Truncate(tariff.SlotDuration)
	rate := func(i int, power float64) api.Rate {
		return api.Rate{
			Start: slot.Add(time.Duration(i) * tariff.SlotDuration),
			End:   slot.Add(time.Duration(i+1) * tariff.SlotDuration),
			Value: power,
		}
	}

	site := new(Site)
	rr := api.Rates{rate(0, 4000), rate(1, 8000)}

	// first slot has no completed predecessor
	assert.Equal(t, 0.0, site.forecastSlotEnergy(rr, slot))

	// repeat ticks within the slot don't re-sample
	assert.Equal(t, 0.0, site.forecastSlotEnergy(rr, slot.Add(time.Minute)))

	// slot 0 completes with the energy sampled at its start, 4kW * 15min
	assert.Equal(t, 1.0, site.forecastSlotEnergy(rr, slot.Add(tariff.SlotDuration)))

	// beyond the forecast horizon slot 1 still completes
	assert.Equal(t, 2.0, site.forecastSlotEnergy(rr, slot.Add(2*tariff.SlotDuration)))
	assert.Equal(t, 0.0, site.forecastSlotEnergy(rr, slot.Add(3*tariff.SlotDuration)))
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
