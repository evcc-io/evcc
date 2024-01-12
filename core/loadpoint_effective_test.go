package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
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
		lpMin, lpMax               float64
		effectiveMin, effectiveMax float64
	}{
		// In this section the charger has a limit set
		{0, 0, 0, 0, 0, 0, 6, 16},    // default
		{1, 10, 0, 0, 0, 0, 1, 10},   // charger lower
		{10, 20, 0, 0, 0, 0, 10, 16}, // charger higher - max ignored
		// In this section the charger and vehicle has a limit
		{0, 0, 1, 10, 0, 0, 6, 10},     // vehicle lower - min ignored
		{0, 0, 10, 20, 0, 0, 10, 16},   // vehicle higher - max ignored
		{1, 10, 2, 12, 0, 0, 2, 10},    // charger + vehicle lower
		{10, 20, 12, 22, 0, 0, 12, 16}, // charger + vehicle higher

		// In this section the charger and vehicle and LP has a limit
		{2, 20, 3, 22, 5, 0, 5, 16},   // lp defines min limit
		{2, 20, 3, 22, 1, 0, 3, 16},   // vehicle defines min limit
		{2, 20, 1, 22, 1, 0, 2, 16},   // charger defines min limit
		{10, 20, 12, 22, 2, 5, 12, 5}, // lp defines max limit
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)
		ctrl := gomock.NewController(t)

		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.charger = api.NewMockCharger(ctrl)
		if tc.lpMax > 0 {
			lp.SetMaxCurrent(tc.lpMax)
		}
		if tc.lpMin > 0 {
			lp.SetMinCurrent(tc.lpMin)
		}

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

		assert.Equal(t, tc.effectiveMin, lp.effectiveMinCurrent())
		assert.Equal(t, tc.effectiveMax, lp.effectiveMaxCurrent())
	}
}
