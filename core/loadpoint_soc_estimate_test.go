package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/soc"
	serverdb "github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSocEstimatePlausible(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	const eps = 964.7 // Wh per point for an 82 kWh vehicle

	tc := []struct {
		name      string
		se        socEstimate
		plausible bool
	}{
		{
			"fresh record",
			socEstimate{AnchorSoc: 15, EnergySinceAnchor: 6800, Updated: now.Add(-time.Hour)},
			true,
		},
		{
			"not updated for more than a day",
			socEstimate{AnchorSoc: 15, EnergySinceAnchor: 6800, Updated: now.Add(-25 * time.Hour)},
			false,
		},
		{
			"offset beyond the cap",
			socEstimate{AnchorSoc: 15, EnergySinceAnchor: 60000, Updated: now.Add(-time.Hour)},
			false,
		},
		{
			"no energy since the anchor",
			socEstimate{AnchorSoc: 15, EnergySinceAnchor: 0, Updated: now.Add(-time.Hour)},
			false,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.plausible, tc.se.plausible(eps, now))
		})
	}
}

func TestUpdateSocEstimateStoresAnchor(t *testing.T) {
	clk := clock.NewMock()
	ctrl := gomock.NewController(t)

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(8.5).AnyTimes()

	lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk}
	estimator := soc.NewEstimator(lp.log, api.NewMockCharger(ctrl), vehicle)

	source := 15.0
	estimator.Soc(&source, 0)   // anchor at 15
	estimator.Soc(&source, 300) // 300 Wh since

	lp.updateSocEstimate(estimator, "test:anchor")

	se, ok := loadSocEstimate("test:anchor")
	require.True(t, ok)
	assert.Equal(t, 15.0, se.AnchorSoc)
	assert.Equal(t, 300.0, se.EnergySinceAnchor)
}

// TestRestoreSocEstimateSurvivesSessionCounterReset drives the connect path in
// production order. evVehicleConnectHandler runs vehicleDefaultOrDetect — and
// with it setActiveVehicle — first, and createSession second; createSession is
// what resets the session energy counter the anchor is measured against.
//
// Restoring in setActiveVehicle therefore anchors against the *previous*
// session's counter, which is zeroed one call later. That looks correct at
// process start, where the counter genuinely is zero, but collapses the
// estimate onto the vehicle's stale reading on every subsequent connect — and
// updateSocEstimate then overwrites the stored record, destroying the history
// rather than just mis-displaying it.
//
// Both halves have to be driven through lp.energyMetrics and lp.createSession
// rather than through ad-hoc numbers, or the test silently models an order
// production never takes.
func TestRestoreSocEstimateSurvivesSessionCounterReset(t *testing.T) {
	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	require.NoError(t, err)

	store, err := session.NewStore("foo", serverdb.Instance)
	require.NoError(t, err)

	clk := clock.NewMock()
	ctrl := gomock.NewController(t)

	mm := api.NewMockMeter(ctrl)
	me := api.NewMockMeterEnergy(ctrl)
	me.EXPECT().TotalEnergy().Return(0.0, nil).AnyTimes()

	type energyDecorator struct {
		api.Meter
		api.MeterEnergy
	}

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clk,
		db:          store,
		chargeMeter: &energyDecorator{Meter: mm, MeterEnergy: me},
	}

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(8.5).AnyTimes()

	require.NoError(t, saveSocEstimate("test:counterreset", socEstimate{
		AnchorSoc: 15, EnergySinceAnchor: 500, Updated: clk.Now(),
	}))
	lp.socEstimateVehicle = "test:counterreset"

	// at connect time the counter still holds the previous session's total;
	// energyMetrics is only ever reset from createSession
	lp.energyMetrics.Update(0.5)
	require.Equal(t, 500.0, lp.GetChargedEnergy())

	// setActiveVehicle builds a fresh estimator
	lp.socEstimator = soc.NewEstimator(lp.log, api.NewMockCharger(ctrl), vehicle)

	// createSession zeroes the counter and re-anchors against it
	lp.createSession()
	require.Equal(t, 0.0, lp.GetChargedEnergy(), "createSession must have reset the counter")

	// the first poll of the new session still sees the stale vehicle value
	source := 15.0
	assert.Equal(t, 20.0, lp.socEstimator.Soc(&source, lp.GetChargedEnergy()),
		"estimate must survive the counter reset, not collapse onto the stale source value")

	// and the record must not have been flattened
	lp.updateSocEstimate(lp.socEstimator, lp.socEstimateVehicle)
	se, ok := loadSocEstimate("test:counterreset")
	require.True(t, ok)
	assert.Equal(t, 500.0, se.EnergySinceAnchor, "history must not be overwritten with zero")
}
