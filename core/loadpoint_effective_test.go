package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEffectiveLimitSoc(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	assert.Equal(t, 100, lp.effectiveLimitSoc())
}

func TestEffectiveMinSoc(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	for _, tc := range []struct {
		loadpoint, vehicle, expected int
	}{
		{0, 0, 0},
		{10, 0, 10},  // loadpoint only
		{0, 20, 20},  // vehicle only
		{10, 20, 20}, // vehicle wins
		{20, 10, 20}, // loadpoint wins
	} {
		t.Logf("%+v", tc)
		config.Reset()

		ctrl := gomock.NewController(t)
		v := api.NewMockVehicle(ctrl)

		const name = "vehicle"
		require.NoError(t, config.Vehicles().Add(
			config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
		))
		settings.SetInt("vehicle."+name+"."+keys.MinSoc, int64(tc.vehicle))

		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.vehicle = v
		lp.minSoc = tc.loadpoint

		assert.Equal(t, tc.expected, lp.effectiveMinSoc())
	}
}

func TestEffectiveMinMaxCurrent(t *testing.T) {
	tc := []struct {
		chargerMin, chargerMax     float64
		vehicleMin, vehicleMax     float64
		effectiveMin, effectiveMax float64
	}{
		{0, 0, 0, 0, 6, 16},
		{2, 0, 0, 0, 2, 16},   // charger min lower, max empty - charger wins
		{7, 0, 0, 0, 7, 16},   // charger min higher, max empty (no practical use)
		{0, 10, 0, 0, 6, 10},  // charger max lower, min empty - loadpoint wins
		{0, 20, 0, 0, 6, 16},  // charger max higher, min empty - loadpoint wins
		{0, 0, 5, 0, 6, 16},   // vehicle min lower, max empty - loadpoint wins
		{0, 0, 8, 0, 8, 16},   // vehicle min higher, max empty - vehicle wins
		{0, 0, 0, 10, 6, 10},  // vehicle max lower, min empty - vehicle wins
		{0, 0, 0, 20, 6, 16},  // vehicle max higher, min empty - loadpoint wins
		{2, 0, 5, 0, 5, 16},   // charger + vehicle min lower, max empty - vehicle wins
		{0, 20, 0, 32, 6, 16}, // charger + vehicle max higher, min empty - loadpoint wins
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)
		ctrl := gomock.NewController(t)

		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.charger = api.NewMockCharger(ctrl)

		if tc.chargerMin+tc.chargerMax > 0 {
			currentLimiter := api.NewMockCurrentLimiter(ctrl)
			currentLimiter.EXPECT().GetMinMaxCurrent().Return(tc.chargerMin, tc.chargerMax, nil).AnyTimes()

			lp.charger = struct {
				api.Charger
				api.CurrentLimiter
			}{
				Charger:        lp.charger,
				CurrentLimiter: currentLimiter,
			}
		}

		if tc.vehicleMin+tc.vehicleMax > 0 {
			vehicle := api.NewMockVehicle(ctrl)
			ac := api.ActionConfig{
				MinCurrent: tc.vehicleMin,
				MaxCurrent: tc.vehicleMax,
			}
			vehicle.EXPECT().OnIdentified().Return(ac).AnyTimes()

			lp.vehicle = vehicle
		}

		assert.Equal(t, tc.effectiveMin, lp.effectiveMinCurrent(), "min")
		assert.Equal(t, tc.effectiveMax, lp.effectiveMaxCurrent(), "max")
	}
}

func TestEffectivePowerLimiter(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	phases := float64(lp.minActivePhases()) // == maxActivePhases for default lp

	powerLimiter := api.NewMockPowerLimiter(ctrl)
	// min 10A, max 12A worth of power across all phases
	powerLimiter.EXPECT().GetMinMaxPower().Return(230*phases*10, 230*phases*12, nil).AnyTimes()

	lp.charger = struct {
		api.Charger
		api.PowerLimiter
	}{
		Charger:      api.NewMockCharger(ctrl),
		PowerLimiter: powerLimiter,
	}

	assert.Equal(t, 10.0, lp.effectiveMinCurrent(), "min")
	assert.Equal(t, 12.0, lp.effectiveMaxCurrent(), "max")
}

// coarse power-limited charger with fixed request must not yield min > max (#31549)
func TestEffectivePowerLimiterCoarse(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	phases := float64(lp.minActivePhases())

	powerLimiter := api.NewMockPowerLimiter(ctrl)
	// fixed 5.5 A/phase request -> fractional, coarse charger truncates to 5 A
	power := 230 * phases * 5.5
	powerLimiter.EXPECT().GetMinMaxPower().Return(power, power, nil).AnyTimes()

	// MockCharger does not implement api.ChargerEx -> coarseCurrent() == true
	lp.charger = struct {
		api.Charger
		api.PowerLimiter
	}{
		Charger:      api.NewMockCharger(ctrl),
		PowerLimiter: powerLimiter,
	}

	minCurrent := lp.effectiveMinCurrent()
	maxCurrent := lp.effectiveMaxCurrent()
	assert.Equal(t, 6.0, minCurrent, "min rounded up to full amps")
	assert.Equal(t, 6.0, maxCurrent, "max rounded up to full amps")
	assert.LessOrEqual(t, minCurrent, maxCurrent, "min must not exceed max")
}

func TestNextPlan(t *testing.T) {
	clock := clock.NewMock()

	ctrl := gomock.NewController(t)
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.charger = api.NewMockCharger(ctrl)

	for _, tc := range []struct {
		planId int
		soc    int
		plans  []plan
	}{
		{1, 0, []plan{
			{Id: 1, End: clock.Now().Add(8 * time.Hour), Soc: 10},
			{Id: 2, End: clock.Now().Add(10 * time.Hour), Soc: 10},
		}},
		{0, 20, []plan{
			{Id: 1, End: clock.Now().Add(8 * time.Hour), Soc: 10},
			{Id: 2, End: clock.Now().Add(10 * time.Hour), Soc: 10},
		}},
		{1, 0, []plan{
			{Id: 1, End: clock.Now().Add(8 * time.Hour), Soc: 20},
			{Id: 2, End: clock.Now().Add(9 * time.Hour), Soc: 20},
		}},
		{2, 0, []plan{
			{Id: 2, End: clock.Now().Add(8 * time.Hour), Soc: 20},
			{Id: 1, End: clock.Now().Add(9 * time.Hour), Soc: 20},
		}},
		{2, 0, []plan{
			{Id: 1, End: clock.Now().Add(8 * time.Hour), Soc: 10},
			{Id: 2, End: clock.Now().Add(10 * time.Hour), Soc: 60},
		}},
		{1, 5, []plan{
			{Id: 1, End: clock.Now().Add(8 * time.Hour), Soc: 10},
			{Id: 2, End: clock.Now().Add(10 * time.Hour), Soc: 20},
		}},
		{2, 15, []plan{
			{Id: 1, End: clock.Now().Add(8 * time.Hour), Soc: 10},
			{Id: 2, End: clock.Now().Add(10 * time.Hour), Soc: 20},
		}},
	} {
		lp.vehicleSoc = float64(tc.soc)

		res := lp.nextActivePlan(1e4, tc.plans)

		if tc.planId == 0 {
			require.Nil(t, res, tc)
			continue
		}

		require.NotNil(t, res, tc)
		assert.Equal(t, tc.planId, res.Id)
	}
}

func TestNextVehiclePlanBaseline(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()

	const name = "test-vehicle"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicle = v

	t.Run("no vehicle plans set", func(t *testing.T) {
		ts, soc, id := lp.nextVehiclePlan()
		assert.True(t, ts.IsZero())
		assert.Equal(t, 0, soc)
		assert.Equal(t, 0, id)
	})

	t.Run("static plan active", func(t *testing.T) {
		targetTime := time.Now().Add(5 * time.Hour).Truncate(time.Second)
		settings.SetTime("vehicle."+name+"."+keys.PlanTime, targetTime)
		settings.SetInt("vehicle."+name+"."+keys.PlanSoc, 80)

		ts, soc, id := lp.nextVehiclePlan()
		assert.Equal(t, 80, soc)
		assert.Equal(t, 1, id)
		assert.WithinDuration(t, targetTime, ts, time.Second)
	})

	t.Run("repeating plan inactive or invalid ignored", func(t *testing.T) {
		settings.SetTime("vehicle."+name+"."+keys.PlanTime, time.Time{})
		settings.SetInt("vehicle."+name+"."+keys.PlanSoc, 0)

		// Set inactive or invalid repeating plans
		rp := []api.RepeatingPlan{
			{
				Weekdays: []int{0, 1, 2, 3, 4, 5, 6},
				Time:     "08:00",
				Tz:       "UTC",
				Soc:      90,
				Active:   false,
			},
			{
				Weekdays: []int{},
				Time:     "08:00",
				Tz:       "UTC",
				Soc:      90,
				Active:   true,
			},
			{
				Weekdays: []int{1},
				Time:     "invalid-time",
				Tz:       "UTC",
				Soc:      90,
				Active:   true,
			},
			{
				Weekdays: []int{1},
				Time:     "08:00",
				Tz:       "Invalid/Timezone",
				Soc:      90,
				Active:   true,
			},
		}
		require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

		ts, soc, id := lp.nextVehiclePlan()
		assert.True(t, ts.IsZero())
		assert.Equal(t, 0, soc)
		assert.Equal(t, 0, id)
	})

	t.Run("active repeating plan returns next occurrence", func(t *testing.T) {
		settings.SetTime("vehicle."+name+"."+keys.PlanTime, time.Time{})
		settings.SetInt("vehicle."+name+"."+keys.PlanSoc, 0)

		// Set active repeating plan
		rp := []api.RepeatingPlan{
			{
				Weekdays: []int{0, 1, 2, 3, 4, 5, 6},
				Time:     "08:00",
				Tz:       "UTC",
				Soc:      85,
				Active:   true,
			},
		}
		require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

		ts, soc, id := lp.nextVehiclePlan()
		assert.Equal(t, 85, soc)
		assert.Equal(t, 2, id)
		assert.False(t, ts.IsZero())
	})
}

func TestPlanLocking(t *testing.T) {
	clk := clock.NewMock()
	now := clk.Now()

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk

	planTime := now.Add(2 * time.Hour)

	t.Run("lock and unlock", func(t *testing.T) {
		lp.lockPlanGoal(planTime, 80, 2)

		// locked values returned before plan target
		ts, soc, id := lp.nextVehiclePlan()
		assert.Equal(t, planTime, ts)
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)

		clk.Add(3 * time.Hour) // advance past plan target

		// locked values persist during overrun
		ts, soc, id = lp.nextVehiclePlan()
		assert.Equal(t, planTime, ts)
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)

		// after clearing, lock is not returned
		lp.clearPlanLock()
		ts, soc, id = lp.nextVehiclePlan()
		assert.True(t, ts.IsZero())
		assert.Equal(t, 0, soc)
		assert.Equal(t, 0, id)
	})
}

func TestGetChargePowerFlexibility(t *testing.T) {
	Voltage = 230

	for _, tc := range []struct {
		mode       api.ChargeMode
		status     api.ChargeStatus
		planActive bool
		want       float64
	}{
		// not charging → always 0
		{api.ModePV, api.StatusB, false, 0},
		// PV mode, charging, no plan → full power is flexible
		{api.ModePV, api.StatusC, false, 2700},
		// PV mode, charging, plan active → not flexible
		{api.ModePV, api.StatusC, true, 0},
		// MinPV mode, charging, no plan → surplus above min is flexible (230V * 6A * 1phase = 1380W)
		{api.ModeMinPV, api.StatusC, false, 2700 - 1380},
		// MinPV mode, charging, plan active → not flexible
		{api.ModeMinPV, api.StatusC, true, 0},
		// Now mode → never flexible, regardless of plan
		{api.ModeNow, api.StatusC, false, 0},
	} {
		t.Run("", func(t *testing.T) {
			lp := NewLoadpoint(util.NewLogger("foo"), nil)
			lp.mode = tc.mode
			lp.status = tc.status
			lp.chargePower = 2700
			lp.planActive = tc.planActive
			// EffectiveMinPower() = 230V * 6A * 1phase = 1380W
			lp.minCurrent = 6
			lp.phases = 1

			assert.Equal(t, tc.want, lp.GetChargePowerFlexibility(nil))
		})
	}
}

// Unit test for T_end <= pausedUntil plan muting
func TestNextVehiclePlanPausedUntilMuting(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()

	const name = "test-vehicle"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	clk := clock.NewMock()
	// T0 = Monday 2026-08-17 00:00:00 UTC
	t0 := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	clk.Set(t0)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.vehicle = v

	// Occurrence #1: Monday 07:00 UTC (T_end), SoC 80%
	// Occurrence #2: Tuesday 07:00 UTC (T_end2), SoC 85%
	rp := []api.RepeatingPlan{
		{
			Weekdays: []int{1}, // Monday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      80,
			Active:   true,
		},
		{
			Weekdays: []int{2}, // Tuesday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      85,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

	// pausedUntil = Monday 2026-08-17 08:00:00 UTC (T_end 07:00 <= pausedUntil 08:00)
	pausedUntil := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	settings.SetTime("vehicle."+name+"."+keys.PausedUntil, pausedUntil)

	// Occurrence #1 (Monday 07:00) is muted and filtered out; Tuesday 07:00 (plan ID = 3) is returned
	ts, soc, id := lp.nextVehiclePlan()
	expectedTuesday := time.Date(2026, 8, 18, 7, 0, 0, 0, time.UTC)
	assert.Equal(t, 85, soc)
	assert.Equal(t, 3, id)
	assert.WithinDuration(t, expectedTuesday, ts, time.Minute)

	// When a single repeating plan exists for Monday, pausing past Monday 07:00 advances to the next week occurrence
	rpSingle := []api.RepeatingPlan{
		{
			Weekdays: []int{1}, // Monday only
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      80,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rpSingle))

	ts, soc, id = lp.nextVehiclePlan()
	expectedNextMonday := time.Date(2026, 8, 24, 7, 0, 0, 0, time.UTC)
	assert.False(t, ts.IsZero())
	assert.Equal(t, 80, soc)
	assert.Equal(t, 2, id)
	assert.WithinDuration(t, expectedNextMonday, ts, time.Minute)
}

// Unit test for T_start < pausedUntil < T_end evaluation (occurrence remains active)
func TestNextVehiclePlanPausedUntilMidPlanEvaluation(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	// Explicitly set max charging power = 10kW so 40kWh (0->80% of 50kWh) requires exactly 4h (T_start = 03:00 UTC)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{MaxPower: 10e3}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()

	const name = "test-vehicle"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	clk := clock.NewMock()
	// T0 = Monday 2026-08-17 04:00:00 UTC
	t0 := time.Date(2026, 8, 17, 4, 0, 0, 0, time.UTC)
	clk.Set(t0)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.vehicle = v
	lp.vehicleSoc = 0

	// Repeating plan: Monday 07:00 UTC, target SoC 80%
	rp := []api.RepeatingPlan{
		{
			Weekdays: []int{1}, // Monday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      80,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

	// pausedUntil = Monday 2026-08-17 05:00:00 UTC (T_start 03:00 < pausedUntil 05:00 < T_end 07:00)
	pausedUntil := time.Date(2026, 8, 17, 5, 0, 0, 0, time.UTC)
	settings.SetTime("vehicle."+name+"."+keys.PausedUntil, pausedUntil)

	// Since T_end (07:00) > pausedUntil (05:00), the plan is NOT muted and remains eligible
	ts, soc, id := lp.nextVehiclePlan()
	expectedEnd := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	assert.Equal(t, 80, soc)
	assert.Equal(t, 2, id)
	assert.WithinDuration(t, expectedEnd, ts, time.Minute)
}

// Unit test for automatic resumption when time passes pausedUntil
func TestNextVehiclePlanAutoResumption(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()

	const name = "test-vehicle"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	clk := clock.NewMock()
	// T0 = Monday 2026-08-17 04:00:00 UTC
	t0 := time.Date(2026, 8, 17, 4, 0, 0, 0, time.UTC)
	clk.Set(t0)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.vehicle = v

	// Repeating plan at 07:00 UTC (id=2, soc=75) and 09:00 UTC (id=3, soc=80)
	rp := []api.RepeatingPlan{
		{
			Weekdays: []int{0, 1, 2, 3, 4, 5, 6}, // every day
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      75,
			Active:   true,
		},
		{
			Weekdays: []int{0, 1, 2, 3, 4, 5, 6}, // every day
			Time:     "09:00",
			Tz:       "UTC",
			Soc:      80,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

	// pausedUntil = in the past relative to mock clock (already expired at 03:00 UTC)
	pausedUntil := t0.Add(-1 * time.Hour)
	settings.SetTime("vehicle."+name+"."+keys.PausedUntil, pausedUntil)

	// Since pausedUntil has elapsed (now >= pausedUntil), isPaused is false and the earliest plan (07:00 UTC) is scheduled
	ts, soc, id := lp.nextVehiclePlan()
	expectedEnd := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	assert.Equal(t, 2, id)
	assert.Equal(t, 75, soc)
	assert.Equal(t, expectedEnd, ts)
}

// Unit test for multi-vehicle isolation: Vehicle A paused does not mute Vehicle B's repeating plans
func TestNextVehiclePlanMultiVehicleIsolation(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)

	// Setup Vehicle A
	vA := api.NewMockVehicle(ctrl)
	vA.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	vA.EXPECT().Phases().Return(0).AnyTimes()
	vA.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	vA.EXPECT().Features().Return(nil).AnyTimes()

	// Setup Vehicle B
	vB := api.NewMockVehicle(ctrl)
	vB.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	vB.EXPECT().Phases().Return(0).AnyTimes()
	vB.EXPECT().Capacity().Return(float64(60)).AnyTimes()
	vB.EXPECT().Features().Return(nil).AnyTimes()

	const nameA = "vehicle-a"
	const nameB = "vehicle-b"

	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: nameA}, api.Vehicle(vA)),
	))
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: nameB}, api.Vehicle(vB)),
	))

	clk := clock.NewMock()
	// T0 = Monday 2026-08-17 00:00:00 UTC
	t0 := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	clk.Set(t0)

	// Repeating plan for Vehicle A: Monday 07:00 UTC, target SoC 80%
	rpA := []api.RepeatingPlan{
		{
			Weekdays: []int{1}, // Monday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      80,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+nameA+"."+keys.RepeatingPlans, rpA))

	// Repeating plan for Vehicle B: Monday 07:00 UTC, target SoC 90%
	rpB := []api.RepeatingPlan{
		{
			Weekdays: []int{1}, // Monday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      90,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+nameB+"."+keys.RepeatingPlans, rpB))

	// Pause Vehicle A until Monday 08:00 UTC (T_end 07:00 <= pausedUntil 08:00)
	pausedUntilA := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	settings.SetTime("vehicle."+nameA+"."+keys.PausedUntil, pausedUntilA)

	// Vehicle B is not paused
	settings.SetTime("vehicle."+nameB+"."+keys.PausedUntil, time.Time{})

	// Loadpoint for Vehicle A
	lpA := NewLoadpoint(util.NewLogger("lpA"), nil)
	lpA.clock = clk
	lpA.vehicle = vA

	// Loadpoint for Vehicle B
	lpB := NewLoadpoint(util.NewLogger("lpB"), nil)
	lpB.clock = clk
	lpB.vehicle = vB

	// 1. Vehicle A's Monday repeating plan is paused, so nextVehiclePlan returns the next occurrence (next Monday 2026-08-24 07:00 UTC)
	tsA, socA, idA := lpA.nextVehiclePlan()
	expectedNextMonday := time.Date(2026, 8, 24, 7, 0, 0, 0, time.UTC)
	assert.False(t, tsA.IsZero(), "vehicle A should advance to next occurrence after pausedUntil")
	assert.Equal(t, 80, socA)
	assert.Equal(t, 2, idA)
	assert.WithinDuration(t, expectedNextMonday, tsA, time.Minute)

	// 2. Vehicle B's repeating plan must NOT be muted and should return this Monday (2026-08-17 07:00 UTC)
	tsB, socB, idB := lpB.nextVehiclePlan()
	expectedB := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	assert.False(t, tsB.IsZero(), "vehicle B repeating plan must not be muted by vehicle A's pausedUntil")
	assert.Equal(t, 90, socB)
	assert.Equal(t, 2, idB)
	assert.WithinDuration(t, expectedB, tsB, time.Minute)

	// 3. Switching loadpoint A's vehicle to Vehicle B should return Vehicle B's active plan
	lpA.vehicle = vB
	tsSwitch, socSwitch, idSwitch := lpA.nextVehiclePlan()
	assert.False(t, tsSwitch.IsZero(), "vehicle B on lpA should not be muted by vehicle A's pause")
	assert.Equal(t, 90, socSwitch)
	assert.Equal(t, 2, idSwitch)
	assert.WithinDuration(t, expectedB, tsSwitch, time.Minute)

	// 4. Pausing Vehicle B should advance Vehicle B to next Monday, while resuming Vehicle A should restore Vehicle A to this Monday
	pausedUntilB := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	settings.SetTime("vehicle."+nameB+"."+keys.PausedUntil, pausedUntilB)
	settings.SetTime("vehicle."+nameA+"."+keys.PausedUntil, time.Time{})

	lpA.vehicle = vA
	tsA2, socA2, idA2 := lpA.nextVehiclePlan()
	expectedAThisMonday := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	assert.False(t, tsA2.IsZero(), "vehicle A should be restored to this Monday after clearing pause")
	assert.Equal(t, 80, socA2)
	assert.Equal(t, 2, idA2)
	assert.WithinDuration(t, expectedAThisMonday, tsA2, time.Minute)

	tsB2, socB2, idB2 := lpB.nextVehiclePlan()
	assert.False(t, tsB2.IsZero(), "vehicle B should advance to next Monday occurrence after pausedUntil")
	assert.Equal(t, 90, socB2)
	assert.Equal(t, 2, idB2)
	assert.WithinDuration(t, expectedNextMonday, tsB2, time.Minute)
}

// Unit test for Daylight Saving Time (DST) switchover and UTC epoch comparison
func TestNextVehiclePlanDaylightSavingTime(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()

	const name = "dst-vehicle"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	berlinLoc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	clk := clock.NewMock()
	lp := NewLoadpoint(util.NewLogger("dst-test"), nil)
	lp.clock = clk
	lp.vehicle = v

	t.Run("Spring DST switchover (CET to CEST) with exact UTC epoch comparison", func(t *testing.T) {
		// Europe/Berlin spring switch: Sunday 2026-03-29 02:00 CET -> 03:00 CEST.
		// Start clock on Saturday night 2026-03-28 22:00:00 UTC (23:00:00 CET)
		t0 := time.Date(2026, 3, 28, 22, 0, 0, 0, time.UTC)
		clk.Set(t0)

		// Recurring plan for Sunday at 07:00 in Europe/Berlin (target: 2026-03-29 07:00:00 +0200 CEST == 2026-03-29 05:00:00 UTC)
		rp := []api.RepeatingPlan{
			{
				Weekdays: []int{0}, // Sunday
				Time:     "07:00",
				Tz:       "Europe/Berlin",
				Soc:      80,
				Active:   true,
			},
		}
		require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

		// Case 1: pausedUntil == target time in UTC (2026-03-29 05:00:00 UTC) -> advances to next Sunday 2026-04-05 07:00:00 CEST (05:00:00 UTC)
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 3, 29, 5, 0, 0, 0, time.UTC))
		ts, soc, id := lp.nextVehiclePlan()
		expectedNextSunday := time.Date(2026, 4, 5, 7, 0, 0, 0, berlinLoc)
		assert.False(t, ts.IsZero(), "plan must advance to next occurrence when target UTC equals pausedUntil UTC")
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedNextSunday.Equal(ts))

		// Case 2: pausedUntil == 1 second before target time in UTC (2026-03-29 04:59:59 UTC) -> T_end > pausedUntil -> this Sunday active
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 3, 29, 4, 59, 59, 0, time.UTC))
		ts, soc, id = lp.nextVehiclePlan()
		expectedTarget := time.Date(2026, 3, 29, 7, 0, 0, 0, berlinLoc)
		assert.False(t, ts.IsZero())
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedTarget.Equal(ts))
		assert.Equal(t, time.Date(2026, 3, 29, 5, 0, 0, 0, time.UTC), ts.UTC())

		// Case 3: pausedUntil == 1 second after target time in UTC (2026-03-29 05:00:01 UTC) -> advances to next Sunday
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 3, 29, 5, 0, 1, 0, time.UTC))
		ts, soc, id = lp.nextVehiclePlan()
		assert.False(t, ts.IsZero(), "plan must advance to next occurrence when target UTC is before pausedUntil UTC")
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedNextSunday.Equal(ts))

		// Case 4: pausedUntil specified in CET timezone offset before switchover (2026-03-29 06:00:00 +0100 CET == 05:00:00 UTC)
		// Should match 05:00:00 UTC exactly and advance to next Sunday
		cetLoc := time.FixedZone("CET", 3600)
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 3, 29, 6, 0, 0, 0, cetLoc))
		ts, soc, id = lp.nextVehiclePlan()
		assert.False(t, ts.IsZero(), "plan must advance to next occurrence when local CET matches target UTC epoch")
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedNextSunday.Equal(ts))
	})

	t.Run("Autumn DST switchover (CEST to CET) with exact UTC epoch comparison", func(t *testing.T) {
		// Europe/Berlin autumn switch: Sunday 2026-10-25 03:00 CEST -> 02:00 CET.
		// Start clock on Saturday night 2026-10-24 21:00:00 UTC (23:00:00 CEST)
		t0 := time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC)
		clk.Set(t0)

		// Recurring plan for Sunday at 07:00 in Europe/Berlin (target: 2026-10-25 07:00:00 +0100 CET == 2026-10-25 06:00:00 UTC)
		rp := []api.RepeatingPlan{
			{
				Weekdays: []int{0}, // Sunday
				Time:     "07:00",
				Tz:       "Europe/Berlin",
				Soc:      85,
				Active:   true,
			},
		}
		require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

		// Case 1: pausedUntil == target time in UTC (2026-10-25 06:00:00 UTC) -> advances to next Sunday 2026-11-01 07:00:00 CET
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 10, 25, 6, 0, 0, 0, time.UTC))
		ts, soc, id := lp.nextVehiclePlan()
		expectedAutumnNextSunday := time.Date(2026, 11, 1, 7, 0, 0, 0, berlinLoc)
		assert.False(t, ts.IsZero())
		assert.Equal(t, 85, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedAutumnNextSunday.Equal(ts))

		// Case 2: pausedUntil == 1 second before target time in UTC (2026-10-25 05:59:59 UTC) -> T_end > pausedUntil -> this Sunday active
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 10, 25, 5, 59, 59, 0, time.UTC))
		ts, soc, id = lp.nextVehiclePlan()
		expectedTarget := time.Date(2026, 10, 25, 7, 0, 0, 0, berlinLoc)
		assert.False(t, ts.IsZero())
		assert.Equal(t, 85, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedTarget.Equal(ts))
		assert.Equal(t, time.Date(2026, 10, 25, 6, 0, 0, 0, time.UTC), ts.UTC())

		// Case 3: pausedUntil == 1 second after target time in UTC (2026-10-25 06:00:01 UTC) -> advances to next Sunday
		settings.SetTime("vehicle."+name+"."+keys.PausedUntil, time.Date(2026, 10, 25, 6, 0, 1, 0, time.UTC))
		ts, soc, id = lp.nextVehiclePlan()
		assert.False(t, ts.IsZero())
		assert.Equal(t, 85, soc)
		assert.Equal(t, 2, id)
		assert.True(t, expectedAutumnNextSunday.Equal(ts))
	})
}

// Unit test for NTP clock step / time correction and immediate auto-resumption
func TestNextVehiclePlanClockStepNTPCorrection(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Capacity().Return(float64(50)).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()

	const name = "ntp-vehicle"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	clk := clock.NewMock()
	// T0 = Monday 2026-08-17 00:00:00 UTC
	t0 := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	clk.Set(t0)

	lp := NewLoadpoint(util.NewLogger("ntp-test"), nil)
	lp.clock = clk
	lp.vehicle = v

	// Repeating plans: Monday 07:00 UTC (SoC 80%), Tuesday 07:00 UTC (SoC 90%)
	rp := []api.RepeatingPlan{
		{
			Weekdays: []int{1}, // Monday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      80,
			Active:   true,
		},
		{
			Weekdays: []int{2}, // Tuesday
			Time:     "07:00",
			Tz:       "UTC",
			Soc:      90,
			Active:   true,
		},
	}
	require.NoError(t, settings.SetJson("vehicle."+name+"."+keys.RepeatingPlans, rp))

	// Pause until Monday 2026-08-17 08:00:00 UTC
	pausedUntil := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	settings.SetTime("vehicle."+name+"."+keys.PausedUntil, pausedUntil)

	t.Run("NTP forward step before pausedUntil", func(t *testing.T) {
		// Clock steps forward +4 hours to 04:00:00 UTC (still < pausedUntil)
		clk.Set(time.Date(2026, 8, 17, 4, 0, 0, 0, time.UTC))

		ts, soc, id := lp.nextVehiclePlan()
		// Monday 07:00 is muted (07:00 <= 08:00), Tuesday 07:00 is returned
		assert.Equal(t, 90, soc)
		assert.Equal(t, 3, id)
		assert.Equal(t, time.Date(2026, 8, 18, 7, 0, 0, 0, time.UTC), ts)
	})

	t.Run("NTP forward step past pausedUntil triggers immediate auto-resumption", func(t *testing.T) {
		// Clock steps forward to 08:00:01 UTC (past pausedUntil)
		clk.Set(time.Date(2026, 8, 17, 8, 0, 1, 0, time.UTC))

		// Without any DB write or API call, isPaused is false immediately
		ts, soc, id := lp.nextVehiclePlan()
		// Monday 07:00 already passed today (8:00:01 > 07:00), next Monday is scheduled or next occurrence is Tuesday 07:00
		assert.False(t, ts.IsZero())
		assert.Equal(t, 90, soc)
		assert.Equal(t, 3, id)
		assert.Equal(t, time.Date(2026, 8, 18, 7, 0, 0, 0, time.UTC), ts)
	})

	t.Run("NTP backward step restores pause condition cleanly", func(t *testing.T) {
		// Clock steps backward to Monday 02:00:00 UTC (< pausedUntil 08:00:00)
		clk.Set(time.Date(2026, 8, 17, 2, 0, 0, 0, time.UTC))

		// System re-evaluates isPaused as true, muting Monday 07:00 and selecting Tuesday 07:00
		ts, soc, id := lp.nextVehiclePlan()
		assert.Equal(t, 90, soc)
		assert.Equal(t, 3, id)
		assert.Equal(t, time.Date(2026, 8, 18, 7, 0, 0, 0, time.UTC), ts)
	})

	t.Run("Large forward jump across days", func(t *testing.T) {
		// Clock steps forward +7 days to next Monday 06:00:00 UTC
		clk.Set(time.Date(2026, 8, 24, 6, 0, 0, 0, time.UTC))

		// Since 2026-08-24 06:00 is well past pausedUntil (2026-08-17 08:00), Monday 07:00 is active today
		ts, soc, id := lp.nextVehiclePlan()
		assert.Equal(t, 80, soc)
		assert.Equal(t, 2, id)
		assert.Equal(t, time.Date(2026, 8, 24, 7, 0, 0, 0, time.UTC), ts)
	})

	t.Run("Rapid clock oscillations evaluate consistently without state corruption", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			if i%2 == 0 {
				clk.Set(time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC))
				ts, soc, id := lp.nextVehiclePlan()
				assert.Equal(t, 90, soc)
				assert.Equal(t, 3, id)
				assert.Equal(t, time.Date(2026, 8, 18, 7, 0, 0, 0, time.UTC), ts)
			} else {
				clk.Set(time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC))
				ts, soc, id := lp.nextVehiclePlan()
				assert.Equal(t, 90, soc)
				assert.Equal(t, 3, id)
				assert.Equal(t, time.Date(2026, 8, 18, 7, 0, 0, 0, time.UTC), ts)
			}
			_ = lp.EffectivePlanSoc()
			_ = lp.EffectivePlanId()
			_ = lp.EffectivePlanTime()
		}
	})
}
