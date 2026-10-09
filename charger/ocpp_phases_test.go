package charger

import (
	"math"
	"testing"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestWattsProfilePhases checks the watts power target falls back to the
// loadpoint's actually active phases (then 3) when the phase switcher never
// set c.phases (issue #30998, #34120).
func TestWattsProfilePhases(t *testing.T) {
	const current = 16.0

	for _, tc := range []struct {
		name       string
		phases     int // c.phases (0 = phase switcher never called)
		lpPhases   int // loadpoint active phases, -1 = no loadpoint
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
				lp.EXPECT().ActivePhases().Return(tc.lpPhases).AnyTimes()
				c.lp = lp
			}

			profile := c.createChargingProfile(current, 0)
			limit := profile.ChargingSchedule.ChargingSchedulePeriod[0].Limit

			require.Equal(t, math.Ceil(230.0*current*float64(tc.wantPhases)/100)*100, limit)
		})
	}
}

// TestWattsProfileRoundUp checks the minimum current isn't missed at grid voltages above 230V
func TestWattsProfileRoundUp(t *testing.T) {
	for _, tc := range []struct {
		phases int
		want   float64
	}{
		{1, 1400}, // 1380W would be 5.97A at 231V
		{3, 4200}, // 4140W would be 5.97A at 231V
	} {
		c := &OCPP{
			cp:     &ocpp.CP{ChargingRateUnit: types.ChargingRateUnitWatts},
			phases: tc.phases,
		}
		profile := c.createChargingProfile(6, 0)
		require.Equal(t, tc.want, profile.ChargingSchedule.ChargingSchedulePeriod[0].Limit)
	}
}
