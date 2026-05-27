package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTotalPriceFallback(t *testing.T) {
	e := embed{Charges: 0.10, Tax: 0.19}
	require.NoError(t, e.init())

	got := e.totalPrice(0.20, time.Now())
	assert.InDelta(t, (0.20+0.10)*1.19, got, 1e-9)
}

func TestTotalPriceMatchingZone(t *testing.T) {
	e := embed{
		Charges: 0.50,
		ChargesZones_: []chargesZoneConfig{
			{Charges: 0.10, Hours: "00:00-06:00"},
		},
	}
	require.NoError(t, e.init())

	ts := time.Date(2026, 1, 15, 2, 0, 0, 0, time.Local)
	got := e.totalPrice(0.20, ts)
	assert.InDelta(t, 0.20+0.10, got, 1e-9)
}

func TestTotalPriceNonMatchingZone(t *testing.T) {
	e := embed{
		Charges: 0.50,
		ChargesZones_: []chargesZoneConfig{
			{Charges: 0.10, Hours: "00:00-06:00"},
		},
	}
	require.NoError(t, e.init())

	ts := time.Date(2026, 1, 15, 12, 0, 0, 0, time.Local)
	got := e.totalPrice(0.20, ts)
	assert.InDelta(t, 0.20+0.50, got, 1e-9)
}

func TestTotalPriceNegativeChargesZone(t *testing.T) {
	e := embed{
		Charges: 0.10,
		ChargesZones_: []chargesZoneConfig{
			{Charges: -0.05, Hours: "10:00-12:00"},
		},
	}
	require.NoError(t, e.init())

	ts := time.Date(2026, 1, 15, 11, 0, 0, 0, time.Local)
	got := e.totalPrice(0.20, ts)
	assert.InDelta(t, 0.20-0.05, got, 1e-9)
}

func TestTotalPriceLastZoneWins(t *testing.T) {
	e := embed{
		ChargesZones_: []chargesZoneConfig{
			{Charges: 0.10, Hours: "00:00-06:00"},
			{Charges: 0.05, Hours: "02:00-04:00"},
		},
	}
	require.NoError(t, e.init())

	// 03:00 is covered by both zones; later entry wins
	ts := time.Date(2026, 1, 15, 3, 0, 0, 0, time.Local)
	assert.InDelta(t, 0.20+0.05, e.totalPrice(0.20, ts), 1e-9)

	// 05:00 is only covered by the broader zone
	ts = time.Date(2026, 1, 15, 5, 0, 0, 0, time.Local)
	assert.InDelta(t, 0.20+0.10, e.totalPrice(0.20, ts), 1e-9)
}

func TestEffectiveChargesMonthFilter(t *testing.T) {
	e := embed{
		Charges: 0.20,
		ChargesZones_: []chargesZoneConfig{
			{Charges: 0.05, Months: "Jan-Mar,Oct-Dec", Hours: "00:00-05:00"},
		},
	}
	require.NoError(t, e.init())

	// February, in window: zone applies
	ts := time.Date(2026, 2, 15, 3, 0, 0, 0, time.Local)
	assert.InDelta(t, 0.05, e.effectiveCharges(ts), 1e-9)

	// June, in hour window but month does not match: fallback
	ts = time.Date(2026, 6, 15, 3, 0, 0, 0, time.Local)
	assert.InDelta(t, 0.20, e.effectiveCharges(ts), 1e-9)
}

func TestTotalPriceFormulaSeesResolvedCharges(t *testing.T) {
	e := embed{
		Charges: 0.10,
		ChargesZones_: []chargesZoneConfig{
			{Charges: 0.30, Hours: "10:00-12:00"},
		},
		Formula: "(price + charges) * 2",
	}
	require.NoError(t, e.init())

	// In zone: charges resolves to 0.30
	ts := time.Date(2026, 1, 15, 11, 0, 0, 0, time.Local)
	assert.InDelta(t, (0.20+0.30)*2, e.totalPrice(0.20, ts), 1e-9)

	// Out of zone: falls back to base 0.10
	ts = time.Date(2026, 1, 15, 14, 0, 0, 0, time.Local)
	assert.InDelta(t, (0.20+0.10)*2, e.totalPrice(0.20, ts), 1e-9)
}

func TestEmbedDecodeChargesZones(t *testing.T) {
	other := map[string]any{
		"charges": 0.20,
		"chargesZones": []map[string]any{
			{"charges": 0.05, "months": "Jan-Mar", "hours": "00:00-05:00"},
			{"charges": 0.30, "hours": "18:00-21:00"},
		},
	}

	var cc struct {
		embed `mapstructure:",squash"`
	}
	require.NoError(t, util.DecodeOther(other, &cc))
	require.NoError(t, cc.embed.init())

	assert.Len(t, cc.ChargesZones_, 2)
	assert.InDelta(t, 0.05, cc.ChargesZones_[0].Charges, 1e-9)
	assert.Equal(t, "Jan-Mar", cc.ChargesZones_[0].Months)
	assert.Len(t, cc.chargesZones, 2)
}
