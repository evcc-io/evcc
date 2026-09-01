package slots

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/stretchr/testify/assert"
)

func TestHorizon(t *testing.T) {
	ts := time.Date(2025, 1, 1, 10, 30, 0, 0, time.Local)
	horizon := Horizon(ts)

	// 48h plus end of day
	assert.Equal(t, time.Date(2025, 1, 3, 23, 59, 59, int(time.Second-time.Nanosecond), time.Local), horizon)

	// before 6:00 the day is not extended
	assert.Equal(t, time.Date(2025, 1, 3, 5, 30, 0, 0, time.Local),
		Horizon(time.Date(2025, 1, 1, 5, 30, 0, 0, time.Local)))

	rates := make(api.Rates, 0, 4*96)
	for slot := ts.Truncate(tariff.SlotDuration); len(rates) < cap(rates); slot = slot.Add(tariff.SlotDuration) {
		rates = append(rates, api.Rate{Start: slot, End: slot.Add(tariff.SlotDuration)})
	}

	// 4 days of slots from 10:30, capped at the last slot of Jan 3rd
	assert.Equal(t, 246, Until(rates, horizon, len(rates)))
	assert.Equal(t, time.Date(2025, 1, 3, 23, 45, 0, 0, time.Local), rates[245].Start)

	// shorter forecast is not extended
	assert.Equal(t, 8, Until(rates, horizon, 8))
}

func TestAsTimestamps(t *testing.T) {
	// now is 10 minutes into a 15-minute slot
	now := time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC)

	// dt[0]=300 means first event is 300s (5min) before end of current slot
	// dt[1..] just mark subsequent slot boundaries
	dt := []int{60 * 5, 60 * 15, 60 * 15}

	got := AsTimestamps(dt, now)

	// current slot 12:00-12:15, first timestamp 12:15 - 5min = 12:10
	assert.Equal(t, []time.Time{
		time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 30, 0, 0, time.UTC),
	}, got)
}

func TestBlendMeasured(t *testing.T) {
	ss := []float64{100, 100, 100, 100, 100, 100}
	BlendMeasured(ss, 200, 4)
	assert.Equal(t, []float64{200, 175, 150, 125, 100, 100}, ss)

	// fewer slots than decay length
	short := []float32{100, 100}
	BlendMeasured(short, 200, 4)
	assert.Equal(t, []float32{200, 175}, short)
}

func TestBlendScale(t *testing.T) {
	ss := []float32{100, 100, 100, 100, 100, 100}
	BlendScale(ss, 2, 4)
	assert.Equal(t, []float32{200, 175, 150, 125, 100, 100}, ss)

	// fewer slots than decay length
	short := []float64{100, 100}
	BlendScale(short, 0.5, 4)
	assert.Equal(t, []float64{50, 62.5}, short)
}
