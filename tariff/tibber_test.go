package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/meter/tibber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTibberChargesZones guards that charge zones alone switch to the energy price base (#34558)
func TestTibberChargesZones(t *testing.T) {
	ts := time.Date(2026, 1, 15, 2, 0, 0, 0, time.Local)
	prices := []tibber.Price{{StartsAt: ts, Total: 0.30, Energy: 0.20}}

	tb := &Tibber{embed: &embed{}}
	require.NoError(t, tb.init())
	assert.InDelta(t, 0.30, tb.rates(prices)[0].Value, 1e-9, "total without charges")

	tb = &Tibber{embed: &embed{
		ChargesZones_: []chargesZoneConfig{{Charges: -0.07, Hours: "00:00-03:00"}},
	}}
	require.NoError(t, tb.init())
	assert.InDelta(t, 0.20-0.07, tb.rates(prices)[0].Value, 1e-9, "energy with zone charges")
}
