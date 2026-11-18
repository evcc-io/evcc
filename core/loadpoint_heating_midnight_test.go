package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeatingSessionEndsAtTheDayBoundary: the day is the session, and the day ends at its own last
// second. The tick that crosses midnight is not the boundary — closing there put yesterday's last
// minutes into today's row and labelled today's session "00:02" (measured 2026-11-01: Brauchwasser
// 00:02:20, FBH 00:02:50, Gewölbe 00:03:20 on the screen, with the operator asking for 23:59:59).
// The counter reading at 00:00:00 is interpolated between the last tick of the old day and the first
// tick of the new one, so the split falls on midnight and not on the poll.
func TestHeatingSessionEndsAtTheDayBoundary(t *testing.T) {
	lp, me, db := visibleHeatingLoadpoint(t)
	cl := lp.clock.(*clock.Mock)

	cet := time.FixedZone("CET", 3600)
	meter := 100.0
	me.EXPECT().TotalEnergy().DoAndReturn(func() (float64, error) { return meter, nil }).AnyTimes()

	// 23:59:00 — the first tick, there is no session yet
	cl.Set(time.Date(2026, 10, 31, 23, 59, 0, 0, cet))
	lp.updateDaySession()
	require.NotNil(t, lp.session)
	assert.Equal(t, time.Date(2026, 10, 31, 23, 59, 0, 0, cet), lp.session.Created,
		"a session that starts mid-day is labelled with its own start")

	// 23:59:30 — nothing happens, the reading becomes the sample for the interpolation
	meter = 100.2
	cl.Set(time.Date(2026, 10, 31, 23, 59, 30, 0, cet))
	lp.updateDaySession()

	// 00:00:30 — the day has turned: 0.1 kWh of the 0.3 kWh delta belongs to November
	meter = 100.3
	cl.Set(time.Date(2026, 11, 1, 0, 0, 30, 0, cet))
	lp.updateDaySession()

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 2, "one row per day")

	oct, nov := sessions[0], sessions[1]
	if nov.Created.Before(oct.Created) {
		oct, nov = nov, oct
	}

	assert.Equal(t, time.Date(2026, 10, 31, 23, 59, 0, 0, cet), oct.Created.In(cet))
	assert.Equal(t, time.Date(2026, 10, 31, 23, 59, 59, 0, cet), oct.Finished.In(cet),
		"the previous day ends at 23:59:59, not at the first tick after midnight")
	require.NotNil(t, oct.MeterStart)
	require.NotNil(t, oct.MeterStop)
	assert.InDelta(t, 100.0, *oct.MeterStart, 1e-9)
	assert.InDelta(t, 100.25, *oct.MeterStop, 1e-9, "the counter is interpolated to midnight")
	assert.InDelta(t, 0.25, oct.ChargedEnergy, 1e-6)

	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, cet), nov.Created.In(cet),
		"the new day is labelled with the day boundary")
	require.NotNil(t, nov.MeterStart)
	assert.InDelta(t, 100.25, *nov.MeterStart, 1e-9, "the new day continues at the same reading")
	assert.Zero(t, nov.ChargedEnergy)
}

// TestHeaterMeterAtFallsBackWithoutHistory: without a previous reading there is nothing to interpolate,
// and a wrong guess is worse than the tick's own value.
func TestHeaterMeterAtFallsBackWithoutHistory(t *testing.T) {
	lp, _, _ := visibleHeatingLoadpoint(t)
	cet := time.FixedZone("CET", 3600)

	nowT := time.Date(2026, 11, 1, 0, 0, 30, 0, cet)
	midnight := time.Date(2026, 11, 1, 0, 0, 0, 0, cet)

	// no sample yet
	assert.InDelta(t, 42, lp.meterAt(midnight, 42, nowT), 1e-9)

	// a counter that went backwards (device replaced) must not be interpolated
	lp.dayReading = energyReading{at: nowT.Add(-30 * time.Second), meter: 50}
	assert.InDelta(t, 42, lp.meterAt(midnight, 42, nowT), 1e-9)

	// a plain ramp: half the time, half the delta (sample half a minute before the boundary)
	lp.dayReading = energyReading{at: nowT.Add(-time.Minute), meter: 41.9}
	assert.InDelta(t, 41.95, lp.meterAt(midnight, 42, nowT), 1e-9)
}
