package core

import (
	"math"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPlanRequiredDurationVehicleRemoved(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicleSoc = 50

	// remove the vehicle while socBasedPlanning reads its capacity, i.e. before the snapshot
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().DoAndReturn(func() float64 {
		lp.vmu.Lock()
		lp.vehicle = nil
		lp.vmu.Unlock()
		return 50
	})
	lp.vehicle = vehicle

	assert.Equal(t, time.Duration(0), lp.GetPlanRequiredDuration(80, 10e3))
}

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

func TestUnsampledSocUnknownCapacity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		capacity float64
		status   api.ChargeStatus
		charged  float64
		wantSoc  float64
		wantOk   bool
	}{
		{name: "unknown capacity while charging", capacity: 0, status: api.StatusC, charged: 5, wantSoc: 0, wantOk: false},
		{name: "negative capacity", capacity: -8.5, status: api.StatusC, charged: 5, wantSoc: 0, wantOk: false},
		{name: "unknown capacity while disconnected still assumes 0%", capacity: 0, status: api.StatusA, charged: 5, wantSoc: 0, wantOk: true},
		{name: "known capacity from energy", capacity: 8.5, status: api.StatusC, charged: 2, wantSoc: 20, wantOk: true},
		{name: "known capacity no energy", capacity: 8.5, status: api.StatusC, charged: 0, wantSoc: 0, wantOk: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			vehicle := api.NewMockVehicle(ctrl)
			vehicle.EXPECT().Capacity().Return(tc.capacity).AnyTimes()

			log := util.NewLogger("foo")
			lp := &Loadpoint{
				log:          log,
				vehicle:      vehicle,
				status:       tc.status,
				socEstimator: soc.NewEstimator(log, vehicle),
			}
			lp.energyMetrics.totalKWh = tc.charged

			got, ok := lp.unsampledSoc(vehicle, lp.socEstimator)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.wantSoc, got)
			assert.False(t, math.IsInf(got, 0), "must not return Inf")
		})
	}
}

func TestNextActivePlanUnknownCapacity(t *testing.T) {
	ctrl := gomock.NewController(t)

	// no usable capacity, stale SoC from a previous session is below the goal
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(0.0).AnyTimes()
	vehicle.EXPECT().Features().AnyTimes()

	log := util.NewLogger("foo")
	lp := &Loadpoint{
		log:          log,
		vehicle:      vehicle,
		status:       api.StatusC,
		vehicleSoc:   10,
		socEstimator: soc.NewEstimator(log, vehicle),
	}
	lp.energyMetrics.totalKWh = 5

	plans := []plan{{Id: 2, Soc: 80, End: time.Now().Add(12 * time.Hour)}}
	p := lp.nextActivePlan(1000, plans)
	require.NotNil(t, p)
	assert.Equal(t, 2, p.Id)
}
