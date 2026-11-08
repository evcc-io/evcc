package core

// configurable phase hysteresis.
//
// phasesuppower/phasesdownpower decouple the 1p↔3p switch thresholds from
// minCurrent/maxCurrent so the band can be narrowed (e.g. down below 3.45kW
// for TWC3 1p efficiency) while keeping a low min current for maximum PV
// yield in 1p mode.
//
// While the loadpoint is charging, a switch decision does NOT switch
// immediately: it starts the 0A safety sequence (see
// loadpoint_phases_safety_test.go). The assertions below therefore check the
// pending target, not an immediate phase change.

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPvScalePhasesHysteresis(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := &struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{
		api.NewMockCharger(ctrl),
		api.NewMockPhaseSwitcher(ctrl),
	}

	// the 0A safety sequence disables the charger at 0A
	charger.MockCharger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()
	charger.MockCharger.EXPECT().MaxCurrent(gomock.Any()).Return(nil).AnyTimes()

	dt := time.Minute

	// TWC3 setup: 2A minimum / 18A maximum
	const (
		minA = 2
		maxA = 18
	)

	tcs := []struct {
		desc               string
		phases             int
		sitePower          float64 // negative = PV surplus, small positive = grid import
		upPower, downPower float64 // hysteresis config, 0 = default
		expectPhases       int     // lp.phases after the call
		expectPending      int     // phaseSwitchTarget after the call (0 = none)
		prepare            func(lp *Loadpoint)
	}{
		// scaling down applies to grid-fed charging (sitePower > 0): with a pure
		// PV surplus the vehicle already runs on self-consumption
		{"default: 3p, 3.5kW above 1.38kW min → stay 3p", 3, 0.1, 0, 0, 3, 0, func(lp *Loadpoint) {
			lp.chargePower = 3500
		}},
		{"default: 3p, 1.0kW below 1.38kW min → down pending", 3, 0.1, 0, 0, 3, 1, func(lp *Loadpoint) {
			lp.chargePower = 1000
			lp.phaseTimer = elapsed
		}},

		// custom down threshold: 3p→1p below 3.45kW, 2A min current preserved
		{"custom down: 3p, 3.5kW above 3.45kW → stay 3p", 3, 0.1, 0, 3450, 3, 0, func(lp *Loadpoint) {
			lp.chargePower = 3500
		}},
		{"custom down: 3p, 3.4kW below 3.45kW → down pending", 3, 0.1, 0, 3450, 3, 1, func(lp *Loadpoint) {
			lp.chargePower = 3400
			lp.phaseTimer = elapsed
		}},
		{"custom down: 3p, 3.4kW below 3.45kW, timer running → stay 3p", 3, 0.1, 0, 3450, 3, 0, func(lp *Loadpoint) {
			lp.chargePower = 3400
			lp.phaseTimer = time.Time{}
		}},

		// custom up threshold: 1p→3p above 3.9kW
		{"custom up: 1p, 3.95kW surplus above 3.9kW → up pending", 1, -3950, 3900, 0, 1, 3, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},
		{"custom up: 1p, 3.85kW surplus below 3.9kW → stay 1p", 1, -3850, 3900, 0, 1, 0, nil},
		{"custom up: 1p, 3.95kW surplus above 3.9kW, timer running → stay 1p", 1, -3950, 3900, 0, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = time.Time{}
		}},

		// safety gate: up threshold below 3p min power (2A × 3 × 230V = 1.38kW)
		// must not scale up when 3p cannot sustain the min current
		{"safety: 1p, up 1kW, 1.2kW surplus below 3p min → stay 1p", 1, -1200, 1000, 0, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			clock := clock.NewMock()
			clock.Add(time.Hour) // avoid time.IsZero

			lp := &Loadpoint{
				log:             util.NewLogger("foo"),
				bus:             evbus.New(), // 0A phase-switch sequence publishes via the bus
				wakeUpTimer:     NewTimer(),
				clock:           clock,
				charger:         charger,
				minCurrent:      minA,
				maxCurrent:      maxA,
				phases:          tc.phases,
				measuredPhases:  tc.phases,
				enabled:         true,
				status:          api.StatusC, // charging: switch decisions go through the 0A sequence
				PhasesUpPower:   tc.upPower,
				PhasesDownPower: tc.downPower,
				Enable: loadpoint.ThresholdConfig{
					Delay: dt,
				},
				Disable: loadpoint.ThresholdConfig{
					Delay: dt,
				},
			}

			if tc.prepare != nil {
				tc.prepare(lp)
			}

			res := lp.pvScalePhases(tc.sitePower, minA, maxA, true)
			require.Equal(t, 0, res, tc.desc)
			require.Equal(t, tc.expectPhases, lp.phases, tc.desc)
			require.Equal(t, tc.expectPending, lp.phaseSwitchTarget, tc.desc)
		})
	}
}
