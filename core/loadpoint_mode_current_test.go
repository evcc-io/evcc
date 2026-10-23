package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestModeConfiguredCurrentPresets(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		mode                         api.ChargeMode
		planActive                   bool
		minSoc                       int
		vehicleSoc                   float64
		minPv, maxPv, minNow, maxNow float64
		wantMin, wantMax             float64
	}{
		{
			name:    "unset presets keep global",
			mode:    api.ModeSmart,
			wantMin: 6, wantMax: 16,
		},
		{
			name:  "smart surplus uses pv presets",
			mode:  api.ModeSmart,
			minPv: 6, maxPv: 10,
			wantMin: 6, wantMax: 10,
		},
		{
			name:  "now mode uses now presets",
			mode:  api.ModeNow,
			minPv: 6, maxPv: 10,
			minNow: 6, maxNow: 16,
			wantMin: 6, wantMax: 16,
		},
		{
			name: "smart with plan active uses now presets",
			mode: api.ModeSmart, planActive: true,
			minPv: 6, maxPv: 10,
			minNow: 8, maxNow: 16,
			wantMin: 8, wantMax: 16,
		},
		{
			name: "smart with minSoc not reached uses now presets",
			mode: api.ModeSmart, minSoc: 20, vehicleSoc: 10,
			minPv: 6, maxPv: 10,
			minNow: 10, maxNow: 16,
			wantMin: 10, wantMax: 16,
		},
		{
			name:    "partial pv max only",
			mode:    api.ModeSmart,
			maxPv:   10,
			wantMin: 6, wantMax: 10,
		},
		{
			name:  "off mode ignores presets",
			mode:  api.ModeOff,
			minPv: 6, maxPv: 10,
			minNow: 8, maxNow: 12,
			wantMin: 6, wantMax: 16,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			lp := NewLoadpoint(util.NewLogger("foo"), nil)
			lp.charger = api.NewMockCharger(ctrl)
			lp.mode = tc.mode
			lp.planActive = tc.planActive
			lp.minSoc = tc.minSoc
			lp.vehicleSoc = tc.vehicleSoc
			lp.minCurrentPv = tc.minPv
			lp.maxCurrentPv = tc.maxPv
			lp.minCurrentNow = tc.minNow
			lp.maxCurrentNow = tc.maxNow

			assert.Equal(t, tc.wantMin, lp.effectiveMinCurrent(), "min")
			assert.Equal(t, tc.wantMax, lp.effectiveMaxCurrent(), "max")
		})
	}
}

func TestSetModeCurrentPresetValidation(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewMemorySettings())
	lp.minCurrent = 6
	lp.maxCurrent = 16

	require.Error(t, lp.SetMaxCurrentPv(20))
	require.Error(t, lp.SetMinCurrentPv(5))
	require.NoError(t, lp.SetMaxCurrentPv(10))
	require.NoError(t, lp.SetMinCurrentPv(6))
	require.Error(t, lp.SetMinCurrentPv(12)) // min > max of pair
	require.NoError(t, lp.SetMinCurrentPv(0))
	require.NoError(t, lp.SetMaxCurrentPv(0))

	require.NoError(t, lp.SetMaxCurrentNow(16))
	require.NoError(t, lp.SetMinCurrentNow(8))
	assert.Equal(t, 8.0, lp.GetMinCurrentNow())
	assert.Equal(t, 16.0, lp.GetMaxCurrentNow())
}
