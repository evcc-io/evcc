package core

import (
	"fmt"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWithinResumeThreshold(t *testing.T) {
	tests := []struct {
		name                   string
		vehicleResumeThreshold int
		limitSoc               int
		vehicleSoc             float64
		expectedWithin         bool
	}{
		{
			name:                   "threshold 0 - always returns false",
			vehicleResumeThreshold: 0,
			limitSoc:               80,
			vehicleSoc:             75,
			expectedWithin:         false,
		},
		{
			name:                   "below threshold range (at resume point)",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             70, // 80 - 10 = 70
			expectedWithin:         false,
		},
		{
			name:                   "just below threshold range",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             69.9,
			expectedWithin:         false,
		},
		{
			name:                   "just in threshold range",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             70.1, // > (80-10), < 80
			expectedWithin:         true,
		},
		{
			name:                   "middle in threshold range",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             75, // > (80-10), < 80
			expectedWithin:         true,
		},
		{
			name:                   "just below limit",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             79.9, // > (80-10), < 80
			expectedWithin:         true,
		},
		{
			name:                   "at limit - returns false",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             80,
			expectedWithin:         false,
		},
		{
			name:                   "above limit - returns false",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             85,
			expectedWithin:         false,
		},
		{
			name:                   "high limit (90%), in range",
			vehicleResumeThreshold: 10,
			limitSoc:               90,
			vehicleSoc:             85, // > (90-10), < 90
			expectedWithin:         true,
		},
		{
			name:                   "high limit (90%), below threshold",
			vehicleResumeThreshold: 10,
			limitSoc:               90,
			vehicleSoc:             80, // 90 - 10 = 80
			expectedWithin:         false,
		},
		{
			name:                   "low limit (60%), in threshold",
			vehicleResumeThreshold: 10,
			limitSoc:               60,
			vehicleSoc:             55, // > (60-10), < 60
			expectedWithin:         true,
		},
		{
			name:                   "low limit (60%), below threshold",
			vehicleResumeThreshold: 10,
			limitSoc:               60,
			vehicleSoc:             50, // 60 - 10 = 50
			expectedWithin:         false,
		},
		{
			name:                   "fractional soc values",
			vehicleResumeThreshold: 10,
			limitSoc:               85,
			vehicleSoc:             82.7, // > (85-10), < 85
			expectedWithin:         true,
		},
		{
			name:                   "zero soc - outside threshold",
			vehicleResumeThreshold: 10,
			limitSoc:               80,
			vehicleSoc:             0,
			expectedWithin:         false,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			lp := NewLoadpoint(util.NewLogger("test"), nil)
			lp.mode = api.ModeMinPV
			lp.status = api.StatusB
			lp.vehicleSoc = tc.vehicleSoc
			lp.limitSoc = tc.limitSoc

			mockVehicle := api.NewMockVehicle(ctrl)
			mockVehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

			// Register vehicle in config
			vehicleName := fmt.Sprintf("testvehicle_%d", i)
			var vehicle api.Vehicle = mockVehicle
			dev := config.NewStaticDevice(config.Named{Name: vehicleName}, vehicle)
			require.NoError(t, config.Vehicles().Add(dev))
			defer config.Vehicles().Delete(vehicleName)

			if tc.vehicleResumeThreshold > 0 {
				settings.SetInt(fmt.Sprintf("vehicle.%s.resumeThreshold", vehicleName), int64(tc.vehicleResumeThreshold))
				defer settings.SetInt(fmt.Sprintf("vehicle.%s.resumeThreshold", vehicleName), 0)
			}

			lp.vehicle = mockVehicle

			result := lp.withinResumeThreshold()
			assert.Equal(t, tc.expectedWithin, result)
		})
	}
}
