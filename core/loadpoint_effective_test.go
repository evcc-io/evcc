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

func TestEffectivePriorityScore(t *testing.T) {
	tc := []struct {
		strategy      api.PriorityStrategy
		basis         api.PriorityBasis
		priority      int
		soc, limitSoc float64
		capacity      float64 // vehicle capacity in kWh, 0 = no/unknown vehicle
		ref           float64 // site-wide reference the gap is normalised against
		expected      float64
	}{
		// none: fractional part is always zero
		{api.PriorityNone, api.PriorityBasisPercent, 0, 50, 0, 0, 100, 0},
		{api.PriorityNone, api.PriorityBasisPercent, 2, 50, 0, 0, 100, 2},
		// soc (percent): lower soc scores higher within the tier
		{api.PrioritySoc, api.PriorityBasisPercent, 0, 20, 0, 0, 100, 0.80},
		{api.PrioritySoc, api.PriorityBasisPercent, 0, 80, 0, 0, 100, 0.20},
		{api.PrioritySoc, api.PriorityBasisPercent, 1, 20, 0, 0, 100, 1.80},
		{api.PrioritySoc, api.PriorityBasisPercent, 0, 100, 0, 0, 100, 0}, // full vehicle: no boost
		{api.PrioritySoc, api.PriorityBasisPercent, 0, 0, 0, 0, 100, 0},   // unknown soc: falls back to plain priority
		// deficit (percent): larger gap to the limit soc scores higher within the tier
		{api.PriorityDeficit, api.PriorityBasisPercent, 0, 50, 80, 0, 100, 0.30},
		{api.PriorityDeficit, api.PriorityBasisPercent, 0, 50, 0, 0, 100, 0.50}, // no limit set -> default 100
		{api.PriorityDeficit, api.PriorityBasisPercent, 0, 90, 80, 0, 100, 0},   // soc above limit: no boost
		{api.PriorityDeficit, api.PriorityBasisPercent, 0, 0, 80, 0, 100, 0},    // unknown soc: falls back to plain priority
		// soc (energy): gap is scaled by capacity -> (100-soc)/100*capacity, normalised by ref
		{api.PrioritySoc, api.PriorityBasisEnergy, 0, 20, 0, 50, 100, 0.40}, // 80% * 50kWh = 40kWh of 100kWh
		{api.PrioritySoc, api.PriorityBasisEnergy, 0, 80, 0, 50, 100, 0.10}, // 20% * 50kWh = 10kWh of 100kWh
		{api.PrioritySoc, api.PriorityBasisEnergy, 0, 20, 0, 25, 100, 0.20}, // 80% * 25kWh = 20kWh of 100kWh
		{api.PrioritySoc, api.PriorityBasisEnergy, 0, 20, 0, 0, 100, 0},     // capacity unknown: no comparable gap
		// deficit (energy): (limitSoc-soc)/100*capacity, normalised by ref
		{api.PriorityDeficit, api.PriorityBasisEnergy, 0, 50, 80, 50, 100, 0.15}, // 30% * 50kWh = 15kWh of 100kWh
		{api.PriorityDeficit, api.PriorityBasisEnergy, 0, 50, 80, 0, 100, 0},     // capacity unknown: no comparable gap
		// Steve's case: a small second car is NOT over-prioritized under the energy basis.
		// Percent basis would rank B (40%) above A (50%); energy basis ranks A (needs 37.5kWh) above B (15kWh).
		{api.PrioritySoc, api.PriorityBasisPercent, 0, 50, 0, 75, 100, 0.50}, // car A, percent
		{api.PrioritySoc, api.PriorityBasisPercent, 0, 40, 0, 25, 100, 0.60}, // car B, percent -> B wins
		{api.PrioritySoc, api.PriorityBasisEnergy, 0, 50, 0, 75, 75, 0.50},   // car A, energy -> A wins
		{api.PrioritySoc, api.PriorityBasisEnergy, 0, 40, 0, 25, 75, 0.20},   // car B, energy
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.priority = tc.priority
		lp.vehicleSoc = tc.soc
		lp.limitSoc = int(tc.limitSoc)

		if tc.capacity > 0 {
			ctrl := gomock.NewController(t)
			vehicle := api.NewMockVehicle(ctrl)
			vehicle.EXPECT().Capacity().Return(tc.capacity).AnyTimes()
			vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
			lp.vehicle = vehicle
		}

		assert.InDelta(t, tc.expected, lp.EffectivePriorityScore(tc.strategy, tc.basis, tc.ref), 1e-9)
	}
}

// a vehicle reporting 0% is indistinguishable from an unknown soc and is deliberately read
// as unknown: it forfeits the sub-ordering boost its gap would earn, scoring like a full
// vehicle rather than ranking first
func TestEffectivePriorityScoreZeroSocReadAsUnknown(t *testing.T) {
	score := func(strategy api.PriorityStrategy, soc float64) float64 {
		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.vehicleSoc = soc
		return lp.EffectivePriorityScore(strategy, api.PriorityBasisPercent, 100)
	}

	for _, strategy := range []api.PriorityStrategy{api.PrioritySoc, api.PriorityDeficit} {
		depleted := score(strategy, 0)

		assert.Zero(t, depleted, "soc 0 must score the bare tier, not the 100pp gap it looks like")
		assert.Equal(t, score(strategy, 100), depleted, "soc 0 ties with a full vehicle")
		assert.Less(t, depleted, score(strategy, 90), "soc 0 loses to any vehicle with a known gap")
	}
}

// the energy basis must keep distinct kWh gaps distinct: a big pack near empty has to
// outrank the same pack half full instead of both saturating the fraction
func TestEffectivePriorityScoreEnergyLargePack(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(200.0).AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

	score := func(soc float64) float64 {
		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.vehicleSoc = soc
		lp.vehicle = vehicle
		return lp.EffectivePriorityScore(api.PrioritySoc, api.PriorityBasisEnergy, 200)
	}

	assert.Greater(t, score(5), score(45), "190kWh gap must outrank 110kWh gap")
}

// the fraction must stay ordered just below the tier boundary: near-empty vehicles must
// not collapse into a tie, and an out-of-range limit soc must not reach the next tier
func TestEffectivePriorityScoreFractionBounds(t *testing.T) {
	score := func(strategy api.PriorityStrategy, prio int, soc float64, limitSoc int) float64 {
		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.priority = prio
		lp.vehicleSoc = soc
		lp.limitSoc = limitSoc
		return lp.EffectivePriorityScore(strategy, api.PriorityBasisPercent, 100)
	}

	// raw fractions 0.998 and 0.992, 0.6pp apart
	assert.Greater(t, score(api.PrioritySoc, 0, 0.2, 0), score(api.PrioritySoc, 0, 0.8, 0), "near-empty vehicles must not tie")

	// limit soc is not range-checked: a 199pp deficit must stay inside its tier, on any tier
	for prio := 0; prio <= 10; prio++ {
		assert.Less(t, score(api.PriorityDeficit, prio, 1, 200), float64(prio+1), "score must stay below the next tier")
	}
}

// heating loadpoints alias temperature as soc, which is no charge level: they get
// the plain tier score without strategy sub-ordering
func TestEffectivePriorityScoreHeating(t *testing.T) {
	ctrl := gomock.NewController(t)

	describer := api.NewMockFeatureDescriber(ctrl)
	describer.EXPECT().Features().Return([]api.Feature{api.Heating}).AnyTimes()

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.priority = 1
	lp.vehicleSoc = 55 // temperature
	lp.charger = struct {
		api.Charger
		api.FeatureDescriber
	}{
		Charger:          api.NewMockCharger(ctrl),
		FeatureDescriber: describer,
	}

	assert.Equal(t, 1.0, lp.EffectivePriorityScore(api.PrioritySoc, api.PriorityBasisPercent, 100))
	assert.Equal(t, 1.0, lp.EffectivePriorityScore(api.PriorityDeficit, api.PriorityBasisPercent, 100))
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
