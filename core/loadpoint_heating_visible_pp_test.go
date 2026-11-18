package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	serverdb "github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// visibleHeatingLoadpoint is a heating device that meters itself and never reports power - the shape of
// every SG-Ready endpoint, and the reason its session has to be persisted by the day logic rather than by
// the charging progress.
func visibleHeatingLoadpoint(t *testing.T) (*Loadpoint, *api.MockMeterEnergy, *session.DB) {
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
	fd.EXPECT().Features().AnyTimes().Return([]api.Feature{api.Heating, api.IntegratedDevice})

	charger := struct {
		api.Charger
		api.FeatureDescriber
	}{Charger: cc, FeatureDescriber: fd}

	me := api.NewMockMeterEnergy(ctrl)
	cm := &ppMeter{Meter: api.NewMockMeter(ctrl), MeterEnergy: me}

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clock.NewMock(),
		db:          db,
		charger:     charger,
		chargeMeter: newChargeMeter(cm),
	}

	return lp, me, db
}

// TestHeatingSessionIsPersistedWhileTheDeviceRests: the day's session must be in the database from the
// first tick of the day — otherwise it is invisible in the UI until the next midnight, because a resting
// heating device draws no power and `publishChargeProgress` (the other writer) never runs.
func TestHeatingSessionIsPersistedWhileTheDeviceRests(t *testing.T) {
	lp, me, db := visibleHeatingLoadpoint(t)

	meter := 7601.11
	me.EXPECT().TotalEnergy().DoAndReturn(func() (float64, error) { return meter, nil }).AnyTimes()

	lp.resetHeatingSession()
	require.NotNil(t, lp.session)

	sessions, err := db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1, "the day's session is listed from the first tick")
	assert.Zero(t, sessions[0].ChargedEnergy)

	// the device ran while nobody watched: the counter moved by 0.28 kWh
	meter = 7601.39
	lp.resetHeatingSession()

	sessions, err = db.Sessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1, "one row per day, not one per tick")
	assert.InDelta(t, 0.28, sessions[0].ChargedEnergy, 1e-6)
}
