package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPlanRequiredDurationUnsampledSoc(t *testing.T) {
	ctrl := gomock.NewController(t)

	// 8.5 kWh capacity => 10 kWh virtual capacity at 85% efficiency
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(8.5).AnyTimes()
	vehicle.EXPECT().Features().AnyTimes()

	log := util.NewLogger("foo")
	lp := &Loadpoint{
		log:          log,
		vehicle:      vehicle,
		vehicleSoc:   30, // stale soc from a previous session
		socEstimator: soc.NewEstimator(log, vehicle),
	}

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

	// a valid soc takes over
	vehicleSoc := 30.0
	lp.socEstimator.Soc(&vehicleSoc, lp.getChargedEnergy())
	assert.Equal(t, 30*time.Minute, lp.getPlanRequiredDuration(35, 1000))
}
