package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// charger selecting 3p from 4500 W on, with a 11 kW maximum
var (
	gapMax1p = 4499.0 / grid.Voltage
	gapMin3p = 4500.0 / (3 * grid.Voltage)
	gapMax3p = 11000.0 / (3 * grid.Voltage)
)

type gapLimiter struct{}

func (gapLimiter) GetPhaseMinMaxCurrent(phases int) (float64, float64, error) {
	if phases == 1 {
		return 6, gapMax1p, nil
	}
	return gapMin3p, gapMax3p, nil
}

// millisCharger avoids rounding to full amps
type millisCharger struct{}

func (millisCharger) MaxCurrentMillis(float64) error { return nil }

type gapCharger struct {
	*api.MockCharger
	*api.MockPhaseSwitcher
	gapLimiter
	millisCharger
}

func newGapCharger(ctrl *gomock.Controller) *gapCharger {
	return &gapCharger{
		MockCharger:       api.NewMockCharger(ctrl),
		MockPhaseSwitcher: api.NewMockPhaseSwitcher(ctrl),
	}
}

func TestEffectiveCurrentLimitsPhaseDependent(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.charger = newGapCharger(ctrl)
	lp.minCurrent = 6
	lp.maxCurrent = 32

	limits := lp.effectiveCurrentLimits()
	assert.Equal(t, 6.0, limits.minCurrent(1), "min 1p")
	assert.Equal(t, gapMax1p, limits.maxCurrent(1), "max 1p")
	assert.Equal(t, gapMin3p, limits.minCurrent(3), "min 3p")
	assert.Equal(t, gapMax3p, limits.maxCurrent(3), "max 3p")

	lp.phases = 1
	assert.Equal(t, 6.0, lp.effectiveMinCurrent(), "active min 1p")
	assert.Equal(t, gapMax1p, lp.effectiveMaxCurrent(), "active max 1p")

	lp.phases = 3
	assert.Equal(t, gapMin3p, lp.effectiveMinCurrent(), "active min 3p")
	assert.Equal(t, gapMax3p, lp.effectiveMaxCurrent(), "active max 3p")

	assert.InDelta(t, 1380, lp.EffectiveMinPower(), 1e-6, "min power")
	assert.InDelta(t, 11000, lp.EffectiveMaxPower(), 1e-6, "max power")

	// fixed 1p is limited by the 1p maximum
	lp.phasesConfigured = 1
	assert.InDelta(t, 4499, lp.EffectiveMaxPower(), 1e-6, "max power fixed 1p")
}

func TestEffectivePowerLimiterPhaseSwitching(t *testing.T) {
	ctrl := gomock.NewController(t)

	powerLimiter := api.NewMockPowerLimiter(ctrl)
	powerLimiter.EXPECT().GetMinMaxPower().Return(1380.0, 11000.0, nil).AnyTimes()

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.maxCurrent = 32
	lp.charger = struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
		api.PowerLimiter
		millisCharger
	}{
		api.NewMockCharger(ctrl),
		api.NewMockPhaseSwitcher(ctrl),
		powerLimiter,
		millisCharger{},
	}

	limits := lp.effectiveCurrentLimits()
	assert.Equal(t, 6.0, limits.minCurrent(1), "min 1p")
	assert.Equal(t, 32.0, limits.maxCurrent(1), "max 1p")
	assert.Equal(t, gapMax3p, limits.maxCurrent(3), "max 3p")
}

// a power derived minimum must not undercut the charger's current minimum
func TestEffectiveMinCurrentCombinesLimiters(t *testing.T) {
	for _, tc := range []struct {
		currentMin, powerMin, expected float64
	}{
		{8, 6 * grid.Voltage, 8},
		{6, 10 * grid.Voltage, 10},
	} {
		ctrl := gomock.NewController(t)

		currentLimiter := api.NewMockCurrentLimiter(ctrl)
		currentLimiter.EXPECT().GetMinMaxCurrent().Return(tc.currentMin, 32.0, nil).AnyTimes()

		powerLimiter := api.NewMockPowerLimiter(ctrl)
		powerLimiter.EXPECT().GetMinMaxPower().Return(tc.powerMin, 0.0, nil).AnyTimes()

		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.phases = 1
		lp.charger = struct {
			api.Charger
			api.CurrentLimiter
			api.PowerLimiter
			millisCharger
		}{
			api.NewMockCharger(ctrl),
			currentLimiter,
			powerLimiter,
			millisCharger{},
		}

		assert.Equal(t, tc.expected, lp.effectiveMinCurrent(), "%+v", tc)
	}
}

func TestPvScalePhasesPhaseDependentLimits(t *testing.T) {
	for _, tc := range []struct {
		desc                   string
		phases                 int
		chargePower, sitePower float64
		res, toPhases          int
	}{
		{"3p, below 3p min", 3, 4400, 100, 1, 1},
		{"3p, above 3p min", 3, 4600, 0, 0, 3},
		{"1p, below 1p max", 1, 4300, -100, 0, 1},
		{"1p, above 1p max and 3p min", 1, 4400, -200, 3, 3},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			charger := newGapCharger(ctrl)

			clck := clock.NewMock()
			clck.Add(time.Hour)

			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				bus:            evbus.New(),
				wakeUpTimer:    NewTimer(),
				clock:          clck,
				charger:        charger,
				minCurrent:     6,
				maxCurrent:     32,
				phases:         tc.phases,
				measuredPhases: tc.phases,
				chargePower:    tc.chargePower,
				status:         api.StatusC,
				enabled:        true,
				phaseTimer:     elapsed,
			}

			if tc.res != 0 {
				if tc.toPhases == 3 {
					charger.MockCharger.EXPECT().MaxCurrent(gomock.Any()).Return(nil).AnyTimes()
				}
				charger.MockPhaseSwitcher.EXPECT().Phases1p3p(tc.toPhases).Return(nil)
			}

			require.Equal(t, tc.res, lp.pvScalePhases(tc.sitePower, lp.effectiveCurrentLimits(), true))
			require.Equal(t, tc.toPhases, lp.phases)
		})
	}
}

func TestPvEnablePhasesPhaseDependentLimits(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		sitePower float64
		toPhases  int
	}{
		{"surplus below 3p min", -4400, 0},
		{"surplus above 3p min", -4600, 3},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			lp := &Loadpoint{
				log:         util.NewLogger("foo"),
				clock:       clock.NewMock(),
				charger:     newGapCharger(ctrl),
				site:        &mockSite{},
				wakeUpTimer: NewTimer(),
				minCurrent:  6,
				maxCurrent:  32,
				phases:      1,
				Enable:      loadpoint.ThresholdConfig{Delay: time.Minute},
				solarShare:  1,
				status:      api.StatusB,
			}

			assert.Equal(t, tc.toPhases, lp.pvEnablePhases(tc.sitePower, 0, lp.effectiveCurrentLimits()))
		})
	}
}
