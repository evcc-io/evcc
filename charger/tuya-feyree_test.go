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

func TestTuyaFeyreeCurrentSetting(t *testing.T) {
	for _, tc := range []struct {
		dps      map[string]any
		expected tuyaFeyreeCurrent
	}{
		{map[string]any{"113": "Max16A", "114": float64(10)}, tuyaFeyreeCurrent{"114", 6, 16}},
		{map[string]any{"113": "Max32A", "115": float64(20)}, tuyaFeyreeCurrent{"115", 6, 32}},
		{map[string]any{"113": "Max50A"}, tuyaFeyreeCurrent{"117", 8, 50}},
		{map[string]any{"116": float64(16)}, tuyaFeyreeCurrent{"116", 8, 40}},
	} {
		res, err := tuyaFeyreeCurrentSetting(tc.dps)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, res)
	}

	_, err := tuyaFeyreeCurrentSetting(map[string]any{})
	assert.Error(t, err)
}

func TestTuyaFeyreeVoltage(t *testing.T) {
	assert.Equal(t, 233.0, tuyaFeyreeVoltage(233))
	assert.Equal(t, 233.0, tuyaFeyreeVoltage(2330))
}

func TestTuyaFeyreeValue(t *testing.T) {
	res, err := tuyaFeyreeValue(map[string]any{"109": float64(37)}, "109")
	assert.NoError(t, err)
	assert.Equal(t, 37.0, res)

	_, err = tuyaFeyreeValue(map[string]any{}, "109")
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}
