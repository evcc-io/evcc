package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestTuyaDepowStatus(t *testing.T) {
	for _, tc := range []struct {
		code     any
		expected api.ChargeStatus
	}{
		{float64(100), api.StatusA}, {float64(101), api.StatusA},
		{float64(200), api.StatusB}, {float64(204), api.StatusB},
		{float64(300), api.StatusC},
		{float64(501), api.StatusNone}, {nil, api.StatusNone},
	} {
		res, err := tuyaDepowStatus(tc.code)
		assert.Equal(t, tc.expected, res, tc.code)
		assert.Equal(t, tc.expected == api.StatusNone, err != nil, tc.code)
	}
}

func TestTuyaDepowLimits(t *testing.T) {
	for _, tc := range []struct {
		dps      map[string]any
		min, max int64
	}{
		{map[string]any{}, 6, 16},
		{map[string]any{"107": "[6, 8, 10, 13, 16]", "152": float64(16)}, 6, 16},
		{map[string]any{"107": "[6, 8, 10, 13, 16]", "152": float64(12)}, 6, 12},
		{map[string]any{"107": "[6, 8, 10, 13, 16, 20, 25, 32]", "152": float64(32)}, 6, 32},
	} {
		minCurrent, maxCurrent := tuyaDepowLimits(tc.dps)
		assert.Equal(t, tc.min, minCurrent, tc.dps)
		assert.Equal(t, tc.max, maxCurrent, tc.dps)
	}
}

func TestTuyaDepowCurrent(t *testing.T) {
	steps := "[6, 8, 10, 13, 16]"
	for _, tc := range []struct {
		dps               map[string]any
		current, expected int64
	}{
		{map[string]any{"107": steps}, 5, 6},
		{map[string]any{"107": steps}, 11, 11},
		{map[string]any{"107": steps}, 12, 12},
		{map[string]any{"107": steps}, 14, 14},
		{map[string]any{"107": steps}, 15, 15},
		{map[string]any{"107": steps}, 32, 16},
		{map[string]any{"107": steps, "152": float64(12)}, 15, 12},
	} {
		assert.Equal(t, tc.expected, tuyaDepowCurrent(tc.dps, tc.current), tc.current)
	}
}
