package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEffectiveLimitSoc(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	assert.Equal(t, 100, lp.effectiveLimitSoc())
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
