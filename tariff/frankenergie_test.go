package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrankEnergieRates(t *testing.T) {
	ts := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	prices := []frankEnergiePrice{{From: ts, Till: ts.Add(15 * time.Minute), MarketPrice: 0.10, AllInPrice: 0.25}}

	fe := &FrankEnergie{embed: &embed{}}
	require.NoError(t, fe.init())
	assert.InDelta(t, 0.25, fe.rates(prices)[0].Value, 1e-9, "import all-in")

	fe = &FrankEnergie{embed: &embed{}, export: true}
	require.NoError(t, fe.init())
	assert.InDelta(t, 0.10, fe.rates(prices)[0].Value, 1e-9, "export market")

	fe = &FrankEnergie{embed: &embed{Charges: -0.01}, export: true}
	require.NoError(t, fe.init())
	assert.InDelta(t, 0.09, fe.rates(prices)[0].Value, 1e-9, "export with fee")

	fe = &FrankEnergie{embed: &embed{Charges: 0.02, Tax: 0.21}}
	require.NoError(t, fe.init())
	assert.InDelta(t, 0.12*1.21, fe.rates(prices)[0].Value, 1e-9, "import from market with charges")
}

func TestFrankEnergieRatesAfter(t *testing.T) {
	ts := time.Date(2026, 10, 14, 23, 0, 0, 0, time.UTC)
	rates := api.Rates{
		{Start: ts, End: ts.Add(time.Hour), Value: 1},
		{Start: ts.Add(time.Hour), End: ts.Add(2 * time.Hour), Value: 2},
	}

	res := ratesAfter(rates, ts.Add(time.Hour))
	require.Len(t, res, 1)
	assert.Equal(t, 2.0, res[0].Value)
	assert.Empty(t, ratesAfter(nil, ts))
}
