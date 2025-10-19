package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetPlanRequiredDuration(t *testing.T) {
	ctrl := gomock.NewController(t)

	Voltage = 230

	v := api.NewMockVehicle(ctrl)
	v.EXPECT().Capacity().AnyTimes().Return(10.0) // 10 kWh
	v.EXPECT().Soc().AnyTimes().Return(0.0, nil)
	v.EXPECT().Features().AnyTimes()

	require.NoError(t, config.Vehicles().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Vehicle(v))))

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicle = v

	d := lp.getPlanRequiredDuration(100*soc.ChargeEfficiency, 1e3)
	require.Equal(t, 10*time.Hour, d)
}

func TestPlanActive(t *testing.T) {
	ctrl := gomock.NewController(t)

	clock := clock.NewMock()
	vehicle.Clock = clock // vehicle settings adapter

	Voltage = 230

	v := api.NewMockVehicle(ctrl)
	v.EXPECT().Capacity().AnyTimes().Return(100.0 / soc.ChargeEfficiency * 10e3 / (3 * Voltage * 16))
	v.EXPECT().Soc().AnyTimes().Return(0.0, nil)
	v.EXPECT().Features().AnyTimes()
	v.EXPECT().OnIdentified().AnyTimes()
	v.EXPECT().Phases().AnyTimes()

	require.NoError(t, config.Vehicles().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Vehicle(v))))

	tariff, err := tariff.NewFixedFromConfig(map[string]any{"price": 1})
	require.NoError(t, err)

	siteVehicles := &vehicles{log: util.NewLogger("foo")}

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clock
	lp.charger = api.NewMockCharger(ctrl)
	lp.planner = planner.New(lp.log, tariff)
	lp.status = api.StatusC // charging
	lp.phases = 3           // fix maxEffectiveCurrent
	lp.vehicle = v

	require.NotZero(t, lp.EffectiveMaxPower())

	validateActive := func(wantActive bool) {
		require.Equal(t, wantActive, lp.plannerActive())
		require.Equal(t, wantActive, lp.planActive)
		require.Zero(t, lp.planTime)
	}
	// active := lp.plannerActive()
	// require.False(t, active)
	// require.Equal(t, active, lp.planActive)
	// require.Zero(t, lp.planTime)

	vv, err := siteVehicles.ByName("test")
	require.NoError(t, err)
	require.NoError(t, vv.SetPlanSoc(lp.clock.Now().Add(time.Hour), 0, 100))

	goal, isSocBased := lp.GetPlanGoal()
	require.Equal(t, 100.0, goal, "goal")
	require.Equal(t, true, isSocBased, "isSocBased")

	maxPower := lp.EffectiveMaxPower()
	requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
	require.LessOrEqual(t, 10*time.Hour, requiredDuration, "requiredDuration")

	// active := lp.plannerActive()
	// require.True(t, active)
	// require.Equal(t, active, lp.planActive)
	// require.Zero(t, lp.planTime)

	// 1 hour before plan time
	validateActive(true)

	// 1 hour after plan time
	clock.Add(2 * time.Hour)
	validateActive(true)
}
