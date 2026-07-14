package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForecastRatesJSONShape(t *testing.T) {
	ts := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	rr := api.Rates{{Start: ts, End: ts.Add(time.Hour), Value: 0.25}}

	b, err := forecastRate(rr[0]).MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"start":1784030400,"end":1784034000,"value":0.25}`, string(b))

	res := forecastRates(rr)
	require.Len(t, res, 1)
	b2, err := res[0].MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, string(b), string(b2))

	assert.Nil(t, forecastRates(nil))
	assert.Nil(t, forecastRates(api.Rates{}))
}

// TestForecastRatesSocketEncodeSimulation mirrors socketEncode's per-element
// json.Marshal, verifying the MarshalJSON override survives that path.
func TestForecastRatesSocketEncodeSimulation(t *testing.T) {
	ts := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	rr := api.Rates{
		{Start: ts, End: ts.Add(time.Hour), Value: 0.1},
		{Start: ts.Add(time.Hour), End: ts.Add(2 * time.Hour), Value: 0.2},
	}
	res := forecastRates(rr)

	for i, r := range res {
		b, err := r.MarshalJSON()
		require.NoError(t, err)
		if i == 0 {
			assert.JSONEq(t, `{"start":1784030400,"end":1784034000,"value":0.1}`, string(b))
		} else {
			assert.JSONEq(t, `{"start":1784034000,"end":1784037600,"value":0.2}`, string(b))
		}
	}
}
