package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	serverdb "github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// heatingCharger meters itself: heating + integrated device, like every SG-Ready endpoint
type heatingCharger struct{}

func (c *heatingCharger) Status() (api.ChargeStatus, error) { return api.StatusB, nil }
func (c *heatingCharger) Enabled() (bool, error)            { return false, nil }
func (c *heatingCharger) Enable(bool) error                 { return nil }
func (c *heatingCharger) MaxCurrent(int64) error            { return nil }
func (c *heatingCharger) Features() []api.Feature {
	return []api.Feature{api.Heating, api.IntegratedDevice}
}

// ppMeter decorates a meter with a total energy counter
type ppMeter struct {
	api.Meter
	api.MeterEnergy
}

func heatingLoadpoint(t *testing.T, features []api.Feature) (*Loadpoint, *api.MockMeterEnergy, *session.DB, *clock.Mock) {
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

	me := api.NewMockMeterEnergy(ctrl)
	cm := &ppMeter{Meter: api.NewMockMeter(ctrl), MeterEnergy: me}

	clck := clock.NewMock()

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clck,
		db:          db,
		charger:     charger,
		chargeMeter: newChargeMeter(cm),
	}

	return lp, me, db, clck
}

// TestHeatingSessionBooksCounterDelta: a heating device books the delta of its own counter, not
// the energy evcc happened to integrate while it had the device switched on
func TestHeatingSessionBooksCounterDelta(t *testing.T) {
	lp, me, _, clck := heatingLoadpoint(t, []api.Feature{api.Heating, api.IntegratedDevice})

	me.EXPECT().TotalEnergy().Return(100.0, nil)
	lp.createSession()
	require.NotNil(t, lp.session)
	require.NotNil(t, lp.session.MeterStart)
	assert.Equal(t, 100.0, *lp.session.MeterStart)

	lp.session.Created = clck.Now()

	// evcc did not charge a single tick, the device ran on its own
	me.EXPECT().TotalEnergy().Return(112.5, nil)
	lp.applyEnergyMetrics(lp.session)

	assert.Equal(t, 112.5, *lp.session.MeterStop)
	assert.Equal(t, 12.5, lp.session.ChargedEnergy)
}

// TestHeatingSessionIgnoresCounterReset: a replaced or reset counter must not book negative energy
func TestHeatingSessionIgnoresCounterReset(t *testing.T) {
	lp, me, _, clck := heatingLoadpoint(t, []api.Feature{api.Heating, api.IntegratedDevice})

	me.EXPECT().TotalEnergy().Return(100.0, nil)
	lp.createSession()
	lp.session.Created = clck.Now()

	me.EXPECT().TotalEnergy().Return(90.0, nil)
	lp.applyEnergyMetrics(lp.session)

	assert.Equal(t, 90.0, *lp.session.MeterStop)
	assert.Equal(t, 0.0, lp.session.ChargedEnergy)
}

// TestNonHeatingSessionKeepsRaterEnergy: everything else keeps the upstream accounting
func TestNonHeatingSessionKeepsRaterEnergy(t *testing.T) {
	lp, me, _, clck := heatingLoadpoint(t, []api.Feature{api.IntegratedDevice})

	me.EXPECT().TotalEnergy().Return(100.0, nil)
	lp.createSession()
	lp.session.Created = clck.Now()

	me.EXPECT().TotalEnergy().Return(112.5, nil)
	lp.applyEnergyMetrics(lp.session)

	assert.Equal(t, 112.5, *lp.session.MeterStop)
	assert.Equal(t, 0.0, lp.session.ChargedEnergy)
}

// TestHeatingSessionIsTheDay: a session spans the day, from the day boundary to the next one,
// whether or not evcc ever switches the device on
func TestHeatingSessionIsTheDay(t *testing.T) {
	lp, me, db, clck := heatingLoadpoint(t, []api.Feature{api.Heating, api.IntegratedDevice})
	me.EXPECT().TotalEnergy().Return(100.0, nil).AnyTimes()

	// no session yet - the first tick of the day creates it
	lp.resetHeatingSession()
	require.NotNil(t, lp.session)
	assert.False(t, lp.session.Created.IsZero())

	created := lp.session.Created
	clck.Add(time.Hour)
	lp.resetHeatingSession()
	assert.Equal(t, created, lp.session.Created, "same day keeps the session")

	// a day later yesterday is closed and the new day starts
	clck.Add(24 * time.Hour)
	lp.resetHeatingSession()
	require.NotNil(t, lp.session)
	assert.Equal(t, clck.Now(), lp.session.Created)

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.False(t, sessions[0].Finished.IsZero())
}

// TestHeatingSessionSurvivesDisconnect: an integrated device reports A whenever it rests. The
// day's session must survive that, where a plain integrated device still clears it
func TestHeatingSessionSurvivesDisconnect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		charger  api.Charger
		keepsSes bool
	}{
		{"heating", &heatingCharger{}, true},
		{"plain integrated device", &integratedDeviceCharger{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
			require.NoError(t, err)

			db, err := session.NewStore("foo", serverdb.Instance)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))
			lp.charger = tc.charger
			lp.db = db
			lp.session = &session.Session{Created: time.Now()}
			cr := api.NewMockChargeRater(ctrl)
			cr.EXPECT().ChargedEnergy().Return(0.0, nil).AnyTimes()
			lp.chargeRater = cr

			lp.evVehicleDisconnectHandler()

			if tc.keepsSes {
				assert.NotNil(t, lp.session, "the day's session must survive a pause")
			} else {
				assert.Nil(t, lp.session, "a plain integrated device clears the session")
			}
		})
	}
}
