package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heatingFeatures is what a self-metering heater reports.
var heatingFeatures = []api.Feature{api.Heating, api.IntegratedDevice}

// TestDaySessionSurvivesARestart: a restart in the middle of the day must continue the day's row, not open a
// second one backdated to midnight. Measured 2026-11-01: evcc restarted twice, and 2026-11-01's FBH day was
// split into three rows all dated 00:00 (ids 194, 203, 212) with 0.43 kWh missing between the meter ranges.
func TestDaySessionSurvivesARestart(t *testing.T) {
	lp, db, _, clck, setMeter := loadpointWith(t, api.StatusB, 5000.0, heatingFeatures)

	cet := time.FixedZone("CET", 3600)
	clck.Set(time.Date(2026, 11, 2, 9, 0, 0, 0, cet))

	// first tick of the day creates the day session at midnight, with the counter of that instant
	lp.updateDaySession()
	require.NotNil(t, lp.session)
	first := *lp.session
	require.False(t, first.Created.IsZero())

	// some consumption, persisted by a later tick
	setMeter(5004.0)
	clck.Set(time.Date(2026, 11, 2, 9, 5, 0, 0, cet))
	lp.updateDaySession()

	// restart: same database, same day, fresh in-memory loadpoint, counter moved on
	clck.Set(time.Date(2026, 11, 2, 9, 40, 0, 0, cet))
	setMeter(5007.0)
	lp2, _, _, _, _ := loadpointWith2(t, db, clck, 5007.0)

	lp2.updateDaySession()

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1, "one row per day: the restart continues the open session")
	assert.Equal(t, first.ID, sessions[0].ID, "the same row, not a new one")
	assert.True(t, sessions[0].Finished.IsZero(), "the day is still running")
	require.NotNil(t, sessions[0].MeterStart)
	assert.InDelta(t, 5000.0, *sessions[0].MeterStart, 1e-9, "the start of the day survives the restart")
	require.NotNil(t, sessions[0].MeterStop)
	assert.InDelta(t, 5007.0, *sessions[0].MeterStop, 1e-9)
	assert.InDelta(t, 7.0, sessions[0].ChargedEnergy, 1e-9)
}

// TestDayStartUsesTheClockLocation: the boundary is midnight in the clock's own timezone, never in whatever
// location a library hands back. Measured 2026-11-01: a development run in +08 wrote 13 rows dated
// "2026-11-02 00:00:00+08:00" (17:00 the previous afternoon) which the UI showed as "yesterday landed on today".
func TestDayStartUsesTheClockLocation(t *testing.T) {
	lp, db, _, clck, _ := loadpointWith(t, api.StatusB, 100.0, heatingFeatures)

	plus8 := time.FixedZone("TEST+08", 8*3600)
	clck.Set(time.Date(2026, 11, 1, 9, 0, 0, 0, plus8))

	lp.updateDaySession()
	require.NotNil(t, lp.session)

	// cross that zone's midnight
	clck.Set(time.Date(2026, 11, 2, 0, 10, 0, 0, plus8))
	lp.updateDaySession()

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	assert.Equal(t, time.Date(2026, 11, 1, 9, 0, 0, 0, plus8), sessions[0].Created.In(plus8),
		"the row starts where the loadpoint did, not backdated")

	_, offset := sessions[0].Finished.Zone()
	assert.Equal(t, 8*3600, offset, "the boundary carries the clock's offset, not a library default")
	assert.Equal(t, time.Date(2026, 11, 1, 23, 59, 59, 0, plus8), sessions[0].Finished.In(plus8),
		"the day ends at that zone's last second")
	assert.Equal(t, time.Date(2026, 11, 2, 0, 0, 0, 0, plus8), sessions[1].Created.In(plus8),
		"and the next day begins at that zone's midnight")
}

// loadpointWith2 rebuilds a loadpoint on an existing session store — the restart case.
func loadpointWith2(t *testing.T, db *session.DB, clck *clock.Mock, meter float64) (*Loadpoint, *session.DB, *api.MockChargeRater, *clock.Mock, func(float64)) {
	t.Helper()

	lp, _, rater, _, _ := loadpointWith(t, api.StatusB, meter, heatingFeatures)
	lp.db = db
	lp.clock = clck

	return lp, db, rater, clck, func(v float64) {}
}
