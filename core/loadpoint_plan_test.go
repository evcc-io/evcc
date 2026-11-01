package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newUnsampledSocLoadpoint(t *testing.T) *Loadpoint {
	ctrl := gomock.NewController(t)

	// 8.5 kWh capacity => 10 kWh virtual capacity at 85% efficiency
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(8.5).AnyTimes()
	vehicle.EXPECT().Features().AnyTimes()

	log := util.NewLogger("foo")
	return &Loadpoint{
		log:          log,
		vehicle:      vehicle,
		status:       api.StatusC,
		vehicleSoc:   40, // stale SoC from a previous session
		socEstimator: soc.NewEstimator(log, vehicle),
	}
}

func TestPlanRequiredDurationUnsampledSoc(t *testing.T) {
	lp := newUnsampledSocLoadpoint(t)

	for _, tc := range []struct {
		charged  float64 // kWh
		expected time.Duration
	}{
		{0, 3*time.Hour + 30*time.Minute},
		{2, time.Hour + 30*time.Minute},
		{3.5, 0},
		{25, 0},
	} {
		lp.energyMetrics.totalKWh = tc.charged
		assert.Equal(t, tc.expected, lp.getPlanRequiredDuration(35, 1000), "charged %.1fkWh", tc.charged)
	}

	// energy of a previous session doesn't count while disconnected
	lp.status = api.StatusA
	assert.Equal(t, 3*time.Hour+30*time.Minute, lp.getPlanRequiredDuration(35, 1000))

	// a valid SoC takes over
	lp.status = api.StatusC
	lp.energyMetrics.totalKWh = 25
	vehicleSoc := 30.0
	lp.socEstimator.Soc(&vehicleSoc, lp.getChargedEnergy())
	assert.Equal(t, 30*time.Minute, lp.getPlanRequiredDuration(35, 1000))
}

func TestNextActivePlanUnsampledSoc(t *testing.T) {
	lp := newUnsampledSocLoadpoint(t)
	plans := []plan{{Id: 2, Soc: 35, End: time.Now().Add(12 * time.Hour)}}

	// stale SoC above the goal must not skip the plan
	p := lp.nextActivePlan(1000, plans)
	require.NotNil(t, p)
	assert.Equal(t, 2, p.Id)

	// goal energy charged
	lp.energyMetrics.totalKWh = 3.5
	assert.Nil(t, lp.nextActivePlan(1000, plans))
}
