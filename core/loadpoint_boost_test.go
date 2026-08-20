package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type mockSite struct {
	site.API
	maxDischargePower *float64
	residualPower     float64
	optimized         int
}

func (m *mockSite) Optimize() {
	m.optimized++
}

func (m *mockSite) GetBatteryMaxDischargePower() *float64 {
	return m.maxDischargePower
}

func (m *mockSite) GetResidualPower() float64 {
	return m.residualPower
}

func TestBoostPower(t *testing.T) {
	Voltage = 230
	lp := &Loadpoint{
		log:          util.NewLogger("lp"),
		status:       api.StatusC,
		batteryBoost: boostStart,
		maxCurrent:   16,
		phases:       3,
	}
	s := &mockSite{}
	lp.site = s

	// No max discharge power limit (nil)
	s.maxDischargePower = nil
	// EffectiveMaxPower will be 230 * 16 * 3 = 11040
	res := lp.boostPower(0)
	assert.Equal(t, 11040.0, res)
	assert.Equal(t, boostContinue, lp.batteryBoost)

	// Discharge power limit is 0W (battery empty)
	s.maxDischargePower = new(float64)
	lp.batteryBoost = boostStart
	res = lp.boostPower(0)
	assert.Equal(t, 0.0, res)

	// With max discharge power limit
	limit5000 := 5000.0
	s.maxDischargePower = &limit5000
	lp.batteryBoost = boostStart
	res = lp.boostPower(0)
	assert.Equal(t, 5000.0, res)
	assert.Equal(t, boostContinue, lp.batteryBoost)

	// boostContinue with limit
	lp.batteryBoost = boostContinue
	s.residualPower = 0
	// delta = math.Max(100, 0) = 100
	// plus EffectiveStepPower = 690
	// delta = 790
	// delta = min(790, max(0, 5000 - 0)) = 790
	// res = 0 + 790 + 0 = 790
	res = lp.boostPower(0)
	assert.Equal(t, 790.0, res)

	// boostContinue at limit
	// delta = min(790, max(0, 5000 - 5000)) = 0
	// res = 5000 + 0 + 0 = 5000
	res = lp.boostPower(5000)
	assert.Equal(t, 5000.0, res)

	// boostContinue over limit
	// delta = min(790, max(0, 5000 - 6000)) = 0
	// res = 6000 + 0 + 0 = 6000
	res = lp.boostPower(6000)
	assert.Equal(t, 6000.0, res)

	// boostStart while battery is charging (negative power)
	// battery charging at 2000W, limit is 5000W
	// max discharge capacity = 5000 - (-2000) = 7000W
	// res = max(0, -2000) + 7000 + 0 = 7000W
	lp.batteryBoost = boostStart
	res = lp.boostPower(-2000)
	assert.Equal(t, 7000.0, res)

	// boostContinue while battery is charging (negative power)
	// limit is 50W (less than the standard 790W delta)
	// without raw negative power, delta would be restricted to 50W
	// with raw negative power (-2000W), headroom is 2050W, so delta is allowed to be 790W
	limit50 := 50.0
	s.maxDischargePower = &limit50
	s.residualPower = 0 // base delta = 100 + 690 = 790
	lp.batteryBoost = boostContinue
	res = lp.boostPower(-2000)
	// res = max(0, -2000) + 790 + 0 = 790W
	assert.Equal(t, 790.0, res)
}

type plainCharger struct {
	api.Charger
}

type phaseSwitchCharger struct {
	api.Charger
}

func (phaseSwitchCharger) Phases1p3p(int) error { return nil }

func TestBoostPowerPhaseSwitchGapBridging(t *testing.T) {
	Voltage = 230
	lp := &Loadpoint{
		log:              util.NewLogger("lp"),
		status:           api.StatusC,
		charger:          phaseSwitchCharger{},
		batteryBoost:     boostContinue,
		minCurrent:       6,
		maxCurrent:       16,
		phases:           1,
		phasesConfigured: 3,
	}
	s := &mockSite{}
	lp.site = s

	// boostContinue on 1p with phase switching: delta must cover the gap
	// between 1p@16A (3680W) and 3p@6A (4140W) = 460W, plus the
	// base delta (100W) and coarse step power (1p: 230W)
	limit := 10000.0
	s.maxDischargePower = &limit
	s.residualPower = 0
	res := lp.boostPower(0)
	// delta = 100 (base) + 230 (step@1p) + 460 (gap) = 790
	// res = 0 + 790 + 0 = 790
	assert.Equal(t, 790.0, res)
	// verify gap alone exceeds the 3p minimum threshold
	// available_power ≈ chargePower(3680) + boostReturn(790) = 4470 > 4140
	assert.Greater(t, Voltage*16+res, Voltage*6*3, "boost must bridge 1p-3p gap")

	// already on 3p: no phase gap added, only base + step
	lp.phases = 3
	res = lp.boostPower(0)
	// delta = 100 + 690 (step@3p) = 790
	assert.Equal(t, 790.0, res)
}

func TestBoostPowerPhaseSwitchGapBridgingExclusions(t *testing.T) {
	Voltage = 230
	limit10k := 10000.0
	now := time.Now()

	for _, tc := range []struct {
		name              string
		charger           api.Charger
		phasesSwitched    time.Time
		maxDischargePower *float64
		circuitPower      float64
		expected          float64
	}{
		{
			name:              "no phase switching",
			charger:           plainCharger{},
			maxDischargePower: &limit10k,
			circuitPower:      10000,
			expected:          330,
		},
		{
			name:              "phase switch not completed",
			charger:           phaseSwitchCharger{},
			phasesSwitched:    now,
			maxDischargePower: &limit10k,
			circuitPower:      10000,
			expected:          330,
		},
		{
			name:              "no max discharge power limit",
			charger:           phaseSwitchCharger{},
			maxDischargePower: nil,
			circuitPower:      10000,
			expected:          330,
		},
		{
			name:              "circuit not allowing 3p",
			charger:           phaseSwitchCharger{},
			maxDischargePower: &limit10k,
			circuitPower:      0,
			expected:          330,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			circuit := api.NewMockCircuit(ctrl)
			circuit.EXPECT().ValidatePower(gomock.Any(), gomock.Any()).Return(tc.circuitPower).AnyTimes()

			lp := &Loadpoint{
				log:              util.NewLogger("lp"),
				status:           api.StatusC,
				charger:          tc.charger,
				batteryBoost:     boostContinue,
				minCurrent:       6,
				maxCurrent:       16,
				phases:           1,
				phasesConfigured: 3,
				phasesSwitched:   tc.phasesSwitched,
				circuit:          circuit,
			}
			s := &mockSite{
				maxDischargePower: tc.maxDischargePower,
			}
			lp.site = s

			res := lp.boostPower(0)
			assert.Equal(t, tc.expected, res)
		})
	}
}

// Relaxing the limit resumes a boost it put on hold, tightening it does not.
// Setting 100 disables the feature and ends an active boost.
func TestSetBatteryBoostLimitResume(t *testing.T) {
	for _, tc := range []struct {
		name       string
		boost      int
		from, to   int
		want       int
		wantUpdate bool
	}{
		{"lowering the limit resumes", boostHold, 50, 30, boostStart, true},
		{"removing the limit ends held boost", boostHold, 50, 100, boostDisabled, true},
		{"removing the limit ends active boost", boostContinue, 50, 100, boostDisabled, true},
		{"raising the limit holds", boostHold, 30, 50, boostHold, true},
		{"enabling a limit holds", boostHold, 100, 50, boostHold, true},
		{"unchanged limit holds", boostHold, 50, 50, boostHold, false},
		{"only a held boost resumes", boostContinue, 50, 30, boostContinue, true},
		{"disabled boost stays disabled", boostDisabled, 50, 30, boostDisabled, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lp := &Loadpoint{
				log:               util.NewLogger("lp"),
				settings:          settings.NewDatabaseSettingsAdapter("foo"),
				lpChan:            make(chan *Loadpoint, 1),
				batteryBoost:      tc.boost,
				batteryBoostLimit: tc.from,
			}

			lp.SetBatteryBoostLimit(tc.to)

			assert.Equal(t, tc.to, lp.batteryBoostLimit)
			assert.Equal(t, tc.want, lp.batteryBoost)
			// a changed limit must act now instead of on the next update tick
			assert.Equal(t, tc.wantUpdate, len(lp.lpChan) == 1)
		})
	}
}
