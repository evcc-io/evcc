package tariff

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/tariff/corrently"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGsiRates(t *testing.T) {
	// co2_g_standard is null since mid-2026, co2_g_oekostrom is the fallback
	var res corrently.Forecast
	require.NoError(t, json.Unmarshal([]byte(`{"forecast":[
		{"co2_g_standard":348,"co2_g_oekostrom":23,"timeframe":{"start":1787652000000,"end":1787655600000}},
		{"co2_g_standard":null,"co2_g_oekostrom":25,"timeframe":{"start":1787655600000,"end":1787659200000}},
		{"co2_g_standard":null,"co2_g_oekostrom":null,"timeframe":{"start":1787659200000,"end":1787662800000}}
	]}`), &res))

	rates := gsiRates(res)
	require.Len(t, rates, 2)
	assert.Equal(t, 348.0, rates[0].Value)
	assert.Equal(t, 25.0, rates[1].Value)
}
