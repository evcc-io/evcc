package charger

import (
	"testing"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestWattsProfilePhases checks the watts power target falls back to loadpoint
// phases (then 3) when the phase switcher never set c.phases (issue #30998).
func TestWattsProfilePhases(t *testing.T) {
	const current = 16.0

	for _, tc := range []struct {
		name       string
		phases     int // c.phases (0 = phase switcher never called)
		lpPhases   int // loadpoint phases, -1 = no loadpoint
		wantPhases int
	}{
		{"switcher set", 3, -1, 3},
		{"from loadpoint 1p", 0, 1, 1},
		{"from loadpoint 3p", 0, 3, 3},
		{"no loadpoint fallback", 0, -1, 3},
		{"loadpoint unknown fallback", 0, 0, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &OCPP{
				cp:     &ocpp.CP{ChargingRateUnit: types.ChargingRateUnitWatts},
				phases: tc.phases,
			}

			if tc.lpPhases >= 0 {
				ctrl := gomock.NewController(t)
				lp := loadpoint.NewMockAPI(ctrl)
				lp.EXPECT().GetPhases().Return(tc.lpPhases).AnyTimes()
				c.lp = lp
			}

			profile := c.createTxDefaultChargingProfile(current)
			limit := profile.ChargingSchedule.ChargingSchedulePeriod[0].Limit

			require.Equal(t, 230.0*current*float64(tc.wantPhases), limit)
		})
	}
}
