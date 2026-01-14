package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWithinResumeThreshold(t *testing.T) {
	tests := []struct {
		name           string
		mode           api.ChargeMode
		status         api.ChargeStatus
		limitSoc       int
		vehicleSoc     float64
		hasVehicle     bool
		expectedWithin bool
		note           string
	}{
		{
			name:           "no vehicle - returns false",
			mode:           api.ModeMinPV,
			status:         api.StatusB,
			limitSoc:       80,
			vehicleSoc:     75,
			hasVehicle:     false,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 when no vehicle",
		},
		{
			name:           "mode PV - returns false",
			mode:           api.ModePV,
			status:         api.StatusB,
			limitSoc:       80,
			vehicleSoc:     76,
			hasVehicle:     true,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 for PV mode",
		},
		{
			name:           "mode Off - returns false",
			mode:           api.ModeOff,
			status:         api.StatusB,
			limitSoc:       80,
			vehicleSoc:     76,
			hasVehicle:     true,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 for Off mode",
		},
		{
			name:           "currently charging - returns false",
			mode:           api.ModeMinPV,
			status:         api.StatusC,
			limitSoc:       80,
			vehicleSoc:     76,
			hasVehicle:     true,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 when charging",
		},
		{
			name:           "limit is 100 - returns false",
			mode:           api.ModeMinPV,
			status:         api.StatusB,
			limitSoc:       100,
			vehicleSoc:     95,
			hasVehicle:     true,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 when limit is 100",
		},
		{
			name:           "limit is 0 (default) - returns false",
			mode:           api.ModeMinPV,
			status:         api.StatusB,
			limitSoc:       0,
			vehicleSoc:     75,
			hasVehicle:     true,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 when limit is 0 (defaults to 100)",
		},
		{
			name:           "threshold 0 (no settings) - returns false",
			mode:           api.ModeMinPV,
			status:         api.StatusB,
			limitSoc:       80,
			vehicleSoc:     75,
			hasVehicle:     true,
			expectedWithin: false,
			note:           "effectiveResumeThreshold returns 0 when vehicle has no threshold configured",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create loadpoint
			lp := NewLoadpoint(util.NewLogger("test"), nil)
			lp.mode = tc.mode
			lp.status = tc.status
			lp.vehicleSoc = tc.vehicleSoc
			lp.limitSoc = tc.limitSoc

			// Set up mock vehicle - it will be wrapped by vehicle.Settings()
			// Note: Mock vehicle without database-backed settings will return 0 for threshold
			if tc.hasVehicle {
				mockVehicle := api.NewMockVehicle(ctrl)
				mockVehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
				lp.vehicle = mockVehicle
			}

			// Call the actual method
			result := lp.withinResumeThreshold()

			assert.Equal(t, tc.expectedWithin, result, tc.note)
		})
	}
}
