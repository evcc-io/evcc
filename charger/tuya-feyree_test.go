package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestTuyaFeyreeStatus(t *testing.T) {
	for _, tc := range []struct {
		state    any
		expected api.ChargeStatus
	}{
		{"no_connet", api.StatusA},
		{"connect", api.StatusB}, {"wait_rfid", api.StatusB}, {"wait_charing", api.StatusB}, {"finish", api.StatusB},
		{"charing", api.StatusC},
		{"error", api.StatusNone}, {nil, api.StatusNone},
	} {
		res, err := tuyaFeyreeStatus(tc.state)
		assert.Equal(t, tc.expected, res, tc.state)
		assert.Equal(t, tc.expected == api.StatusNone, err != nil, tc.state)
	}
}

func TestTuyaFeyreeCurrentDp(t *testing.T) {
	for _, tc := range []struct {
		dps      map[string]any
		expected string
	}{
		{map[string]any{"113": "Max16A", "114": float64(10)}, "114"},
		{map[string]any{"113": "Max32A", "115": float64(20)}, "115"},
		{map[string]any{"113": "Max50A"}, "117"},
		{map[string]any{"116": float64(16)}, "116"},
	} {
		res, err := tuyaFeyreeCurrentDp(tc.dps)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, res)
	}

	_, err := tuyaFeyreeCurrentDp(map[string]any{})
	assert.Error(t, err)
}

func TestTuyaFeyreeVoltage(t *testing.T) {
	assert.Equal(t, 233.0, tuyaFeyreeVoltage(233))
	assert.Equal(t, 233.0, tuyaFeyreeVoltage(2330))
}
