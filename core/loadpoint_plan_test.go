package core

import (
	"fmt"
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

	log := util.NewLogger("foo")
	log.SetLogOutput(t.Output())

	siteVehicles := &vehicles{log: log}

	lp := NewLoadpoint(log, nil)
	lp.clock = clock
	lp.charger = api.NewMockCharger(ctrl)
	lp.planner = planner.New(lp.log, tariff, planner.WithClock(clock))
	lp.status = api.StatusC // charging
	lp.phases = 3           // fix maxEffectiveCurrent
	lp.vehicle = v

	require.NotZero(t, lp.EffectiveMaxPower())

	testActive := func(t *testing.T, name string, wantActive bool, foo ...any) {
		t.Run(name, func(t *testing.T) {
			if len(foo) > 0 {
				fmt.Println("foo")
			}

			t.Log("lp.EffectivePlanTime", lp.EffectivePlanTime())
			require.Equal(t, wantActive, lp.plannerActive())
			require.Equal(t, wantActive, lp.planActive)

			if len(foo) > 0 {
				t.Fail()
			}
		})
	}

	vv, err := siteVehicles.ByName("test")
	require.NoError(t, err)

	// no plan
	testActive(t, "no plan", false)

	t.Run("static plan", func(t *testing.T) {
		// set plan
		require.NoError(t, vv.SetPlanSoc(lp.clock.Now().Add(time.Hour), 0, 100))

		goal, isSocBased := lp.GetPlanGoal()
		require.Equal(t, 100.0, goal, "goal")
		require.Equal(t, true, isSocBased, "isSocBased")

		maxPower := lp.EffectiveMaxPower()
		requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
		require.LessOrEqual(t, 10*time.Hour, requiredDuration, "requiredDuration")

		// 1 hour before plan time
		testActive(t, "1 hour before plan time", true)

		// 1 hour after plan time
		clock.Add(2 * time.Hour)
		testActive(t, "1 hour after plan time", true)
	})

	// delete plan, reset time
	clock.Add(-2 * time.Hour)
	require.NoError(t, vv.SetPlanSoc(time.Time{}, 0, 0))

	t.Run("repeating plan", func(t *testing.T) {
		// set plan
		require.NoError(t, vv.SetRepeatingPlans([]api.RepeatingPlanStruct{
			{
				Weekdays: []int{0, 1, 2, 3, 4, 5, 6},
				Time:     "01:00",
				Tz:       clock.Now().Location().String(),
				Soc:      100,
				Active:   true,
			},
		}))

		// 1 hour before plan time
		testActive(t, "1 hour before plan time", true)

		// 1 hour after plan time
		clock.Add(2 * time.Hour)
		testActive(t, "1 hour after plan time", true)
	})
}
