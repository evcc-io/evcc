package tariff

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalVoltsRates(t *testing.T) {
	// three 5-minute import intervals of one slot plus a fourth in the next slot, export interleaved
	const body = `[
		{"NMI":"4100000000","circuit":"Import","direction":"Buy","intervalEnd":"2026-10-19T00:05:00Z","intervalDuration":5,"flexUp":30,"flexUnits":"c/kWh","quality":"Exp"},
		{"NMI":"4100000000","circuit":"Export","direction":"Sell","intervalEnd":"2026-10-19T00:05:00Z","intervalDuration":5,"flexUp":5,"flexUnits":"c/kWh","quality":"Exp"},
		{"NMI":"4100000000","circuit":"Import","direction":"Buy","intervalEnd":"2026-10-19T00:10:00Z","intervalDuration":5,"flexUp":40,"flexUnits":"c/kWh","quality":"Fcst"},
		{"NMI":"4100000000","circuit":"Import","direction":"Buy","intervalEnd":"2026-10-19T00:15:00Z","intervalDuration":5,"flexUp":50,"flexUnits":"c/kWh","quality":"Fcst"},
		{"NMI":"4100000000","circuit":"Import","direction":"Buy","intervalEnd":"2026-10-19T00:20:00Z","intervalDuration":5,"flexUp":-10,"flexUnits":"c/kWh","quality":"Fcst"}
	]`

	var res []localVoltsInterval
	require.NoError(t, json.Unmarshal([]byte(body), &res))

	tr := &LocalVolts{embed: &embed{}, channel: "Import"}
	rates := tr.rates(res)

	require.Len(t, rates, 2)
	assert.Equal(t, time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC).Local(), rates[0].Start)
	assert.Equal(t, SlotDuration, rates[0].End.Sub(rates[0].Start))
	assert.InDelta(t, 0.4, rates[0].Value, 1e-9)
	assert.InDelta(t, -0.1, rates[1].Value, 1e-9)

	tr.channel = "Export"
	rates = tr.rates(res)
	require.Len(t, rates, 1)
	assert.InDelta(t, 0.05, rates[0].Value, 1e-9)
}
