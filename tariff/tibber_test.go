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

func TestTibberRefreshDelay(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	at := func(hour, minute int) time.Time {
		return time.Date(2026, 10, 25, hour, minute, 0, 0, loc)
	}

	for _, tc := range []struct {
		name     string
		now      time.Time
		tomorrow bool
		expected time.Duration
	}{
		{"tomorrow available", at(14, 0), true, time.Hour},
		{"morning", at(9, 0), false, time.Hour},
		{"shortly before publication", at(12, 20), false, 40 * time.Minute},
		{"waiting for publication", at(13, 10), false, 5 * time.Minute},
		{"evening without tomorrow", at(20, 0), false, 5 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tibberRefreshDelay(tc.now, "Europe/Berlin", tc.tomorrow))
		})
	}
}

func TestTibberRefreshDelayTimeZone(t *testing.T) {
	// 12:30 UTC is 13:30 in Oslo during winter time, so prices are due there
	now := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	assert.Equal(t, 5*time.Minute, tibberRefreshDelay(now, "Europe/Oslo", false))
}
