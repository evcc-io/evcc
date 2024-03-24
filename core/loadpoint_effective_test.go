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
