package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestPvScalePhasesSmartLimit asserts that a configured solar 1p limit keeps
// surplus charging on a single phase, even with ample surplus that would
// otherwise scale up to 3 phases. Fast charging paths never reach
// pvScalePhases, so they are unaffected by construction.
func TestPvScalePhasesSmartLimit(t *testing.T) {
	Voltage = 230

	for _, limited := range []bool{false, true} {
		ctrl := gomock.NewController(t)

		plainCharger := api.NewMockCharger(ctrl)
		phaseCharger := api.NewMockPhaseSwitcher(ctrl)

		vehicle := api.NewMockVehicle(ctrl)
		vehicle.EXPECT().Phases().Return(3).AnyTimes()
		vehicle.EXPECT().Features().AnyTimes()

		lp := &Loadpoint{
			log:              util.NewLogger("foo"),
			clock:            clock.NewMock(),
			wakeUpTimer:      NewTimer(),
			minCurrent:       6,
			maxCurrent:       16,
			vehicle:          vehicle,
			phasesConfigured: 0, // allow switching
			phases:           1,
			chargePower:      1 * 6 * Voltage,
			limitSmartPhases: limited,
			status:           api.StatusC,
			charger: struct {
				*api.MockCharger
				*api.MockPhaseSwitcher
			}{plainCharger, phaseCharger},
		}

		// ample surplus for 3p charging
		const sitePower = -10000.0

		if limited {
			// no Phases1p3p expectation: any switch attempt fails the test
			require.Equal(t, 0, lp.pvScalePhases(sitePower, 6, 16, true))
			require.Equal(t, 1, lp.GetPhases())
		} else {
			phaseCharger.EXPECT().Phases1p3p(3).Return(nil)
			require.Equal(t, 3, lp.pvScalePhases(sitePower, 6, 16, true))
			require.Equal(t, 3, lp.GetPhases())
		}

		ctrl.Finish()
	}
}
