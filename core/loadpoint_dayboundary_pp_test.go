package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	serverdb "github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// plainLoadpoint is a loadpoint that is *not* a self-metering device: an ordinary charge point whose energy
// comes either from evcc own integration of the charge power or from a charge meter. That is the case the
// day boundary has to cover as well: a socket heater or a split unit has to land on "that very day" just
// like a heat pump.
func plainLoadpoint(t *testing.T, status api.ChargeStatus, meter float64) (*Loadpoint, *session.DB, *api.MockChargeRater, *clock.Mock, func(float64)) {
	t.Helper()

	return loadpointWith(t, status, meter, nil)
}

// loadpointWith is plainLoadpoint with an explicit feature set: with api.Heating + api.IntegratedDevice the
// same fixture is a self-metering device, which books its counter instead of evcc own accumulator.
func loadpointWith(t *testing.T, status api.ChargeStatus, meter float64, features []api.Feature) (*Loadpoint, *session.DB, *api.MockChargeRater, *clock.Mock, func(float64)) {
	t.Helper()

	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	require.NoError(t, err)

	db, err := session.NewStore("foo", serverdb.Instance)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	cc := api.NewMockCharger(ctrl)
	fd := api.NewMockFeatureDescriber(ctrl)
	fd.EXPECT().Features().AnyTimes().Return(features)

	charger := struct {
		api.Charger
		api.FeatureDescriber
	}{Charger: cc, FeatureDescriber: fd}

	rater := api.NewMockChargeRater(ctrl)

	// charge meter counter, mutable from the test
	m := meter
	me := api.NewMockMeterEnergy(ctrl)
	me.EXPECT().TotalEnergy().DoAndReturn(func() (float64, error) { return m, nil }).AnyTimes()
	cm := &ppMeter{Meter: api.NewMockMeter(ctrl), MeterEnergy: me}

	clck := clock.NewMock()
	clck.Set(time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC))

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clck,
		db:          db,
		charger:     charger,
		chargeMeter: newChargeMeter(cm),
		chargeRater: rater,
		status:      status,
	}

	setMeter := func(v float64) { m = v }

	return lp, db, rater, clck, setMeter
}

// TestSessionIsCutAtTheDayBoundaryForAnyLoadpoint: the day is the accounting unit, not a charging run. A
// session still open at midnight ends on 23:59:59 with the energy of that instant — interpolated, because
// the boundary is not a tick — and continues on 00:00:00, so nothing consumed on one day is booked to the
// next. Measured reason (11/2026-11-01): a socket heater or a split running past midnight carried yesterday
// kWh into today row, and evcc only ever wrote one row for the whole run.
func TestSessionIsCutAtTheDayBoundaryForAnyLoadpoint(t *testing.T) {
	lp, db, rater, clck, setMeter := plainLoadpoint(t, api.StatusC, 100.0)

	cet := time.FixedZone("CET", 3600)
	start := time.Date(2026, 11, 1, 23, 0, 0, 0, cet)

	clck.Set(start)
	lp.createSessionAt(100.0)
	require.NotNil(t, lp.session)
	lp.session.Created = start

	// 23:59:30 — 2 kWh consumed, first reading sample
	clck.Set(time.Date(2026, 11, 1, 23, 59, 30, 0, cet))
	lp.energyMetrics.Update(2.0)
	lp.updateDaySession()

	// 00:00:30 — 3 kWh, still charging: the day turned with the session open
	clck.Set(time.Date(2026, 11, 2, 0, 0, 30, 0, cet))
	lp.energyMetrics.Update(3.0)
	setMeter(100.3)
	rater.EXPECT().ChargedEnergy().Return(3.0, nil)
	lp.updateDaySession()

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 2, "one row per day, not one per charging run")

	first, second := sessions[0], sessions[1]
	if second.Created.Before(first.Created) {
		first, second = second, first
	}

	assert.Equal(t, start, first.Created.In(cet))
	assert.Equal(t, time.Date(2026, 11, 1, 23, 59, 59, 0, cet), first.Finished.In(cet),
		"the previous day ends at its last second")
	assert.InDelta(t, 2.5, first.ChargedEnergy, 1e-6, "half of the 1 kWh between the ticks belongs to the old day")
	require.NotNil(t, first.MeterStop)
	assert.InDelta(t, 100.15, *first.MeterStop, 1e-9, "the counter is interpolated to midnight as well")

	require.NotNil(t, lp.session, "still charging: the new day continues the session")
	assert.Equal(t, second.Created.In(cet), lp.session.Created.In(cet))
	assert.Equal(t, time.Date(2026, 11, 2, 0, 0, 0, 0, cet), lp.session.Created.In(cet))
	require.NotNil(t, lp.session.MeterStart)
	assert.InDelta(t, 100.15, *lp.session.MeterStart, 1e-9)
	assert.Zero(t, lp.session.ChargedEnergy, "the new day starts empty")
	assert.InDelta(t, 3.0, lp.chargedAtStartup, 1e-9, "the rater offset follows the boundary")
	assert.Zero(t, lp.energyMetrics.TotalWh(), "the accumulator restarts for the new day")
}

// TestSessionEndsAtTheBoundaryWhenIdle: nothing is running at midnight, so the day simply ends there and no
// empty session is invented for the new day.
func TestSessionEndsAtTheBoundaryWhenIdle(t *testing.T) {
	lp, db, _, clck, _ := plainLoadpoint(t, api.StatusB, 0)

	cet := time.FixedZone("CET", 3600)
	start := time.Date(2026, 11, 1, 22, 0, 0, 0, cet)

	clck.Set(start)
	lp.createSession()
	require.NotNil(t, lp.session)
	lp.session.Created = start
	lp.energyMetrics.Update(4.0)

	clck.Set(time.Date(2026, 11, 1, 23, 58, 0, 0, cet))
	lp.updateDaySession()
	clck.Set(time.Date(2026, 11, 2, 0, 1, 0, 0, cet))
	lp.updateDaySession()

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, time.Date(2026, 11, 1, 23, 59, 59, 0, cet), sessions[0].Finished.In(cet))
	assert.Nil(t, lp.session, "no continuation without a load")
	assert.Zero(t, lp.energyMetrics.TotalWh())
}

// TestDayBoundaryKeepsItsShareOfTheCost: with a tariff the closed part must carry its own cost, not the
// whole session's — otherwise a heater that runs past midnight would book the evening twice.
func TestDayBoundaryKeepsItsShareOfTheCost(t *testing.T) {
	lp, _, rater, clck, _ := plainLoadpoint(t, api.StatusC, 0)

	cet := time.FixedZone("CET", 3600)
	price := 0.33
	lp.energyMetrics.SetEnvironment(0, &price, nil)

	clck.Set(time.Date(2026, 11, 1, 23, 0, 0, 0, cet))
	lp.createSession()
	lp.session.Created = time.Date(2026, 11, 1, 23, 0, 0, 0, cet)

	clck.Set(time.Date(2026, 11, 1, 23, 59, 30, 0, cet))
	lp.energyMetrics.Update(2.0)
	lp.updateDaySession()

	clck.Set(time.Date(2026, 11, 2, 0, 0, 30, 0, cet))
	lp.energyMetrics.Update(4.0)
	rater.EXPECT().ChargedEnergy().Return(4.0, nil)
	lp.updateDaySession()

	sessions, err := dbSessions(lp)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	first := sessions[0]
	if first.Created.After(sessions[1].Created) {
		first = sessions[1]
	}

	assert.InDelta(t, 3.0, first.ChargedEnergy, 1e-6, "three quarters of the 4 kWh were before midnight")
	require.NotNil(t, first.Price)
	assert.InDelta(t, 3.0*price, *first.Price, 1e-6, "and three quarters of the cost")
	assert.InDelta(t, price, *first.PricePerKWh, 1e-9)
}

// dbSessions is a shortcut for the store behind a loadpoint.
func dbSessions(lp *Loadpoint) (session.Sessions, error) {
	return lp.db.Sessions()
}

// TestHeatingDayBoundaryKeepsItsShareOfTheCost: a self-metering device books the counter, but its cost comes
// from the accumulator, which at the tick after midnight already carries the new day. The closed row must
// take only its own share, or a heat pump would pay for the morning in yesterday's row.
func TestHeatingDayBoundaryKeepsItsShareOfTheCost(t *testing.T) {
	lp, db, _, clck, setMeter := loadpointWith(t, api.StatusB, 100.0, []api.Feature{api.Heating, api.IntegratedDevice})

	cet := time.FixedZone("CET", 3600)
	price := 0.33
	lp.energyMetrics.SetEnvironment(0, &price, nil)

	// 23:00 — the day session starts at the counter reading of its day
	clck.Set(time.Date(2026, 11, 1, 23, 0, 0, 0, cet))
	lp.updateDaySession()
	require.NotNil(t, lp.session)
	require.NotNil(t, lp.session.MeterStart)
	assert.InDelta(t, 100.0, *lp.session.MeterStart, 1e-9)

	// 23:59:30 — 2 kWh so far, and the reading becomes the sample for the interpolation
	setMeter(102.0)
	clck.Set(time.Date(2026, 11, 1, 23, 59, 30, 0, cet))
	lp.updateDaySession()
	require.NotNil(t, lp.session.MeterStop)
	assert.InDelta(t, 2.0, lp.session.ChargedEnergy, 1e-6)

	// 00:00:30 — 4 kWh; three of them were November the 1st
	setMeter(104.0)
	clck.Set(time.Date(2026, 11, 2, 0, 0, 30, 0, cet))
	lp.updateDaySession()

	all, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, all, 2, "the day session is cut at the boundary and continued")

	first := all[0]
	if all[1].Created.Before(all[0].Created) {
		first = all[1]
	}

	assert.Equal(t, time.Date(2026, 11, 1, 23, 59, 59, 0, cet), first.Finished.In(cet))
	assert.InDelta(t, 3.0, first.ChargedEnergy, 1e-6, "the counter is split at midnight (100 -> 103)")
	require.NotNil(t, first.Price)
	assert.InDelta(t, 3.0*price, *first.Price, 1e-6, "and the cost follows that share, not the whole morning")
	assert.InDelta(t, price, *first.PricePerKWh, 1e-9)
}
