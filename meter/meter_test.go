package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPV(t *testing.T) {
	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
		"maxacpower": 1000,
	})
	require.NoError(t, err)

	// must not have soc/capacity
	_, ok := api.Cap[api.MaxACPowerGetter](m)
	assert.True(t, ok, "MaxACPowerGetter")
}

func TestPVMaxACPowerPlugin(t *testing.T) {
	power := map[string]any{
		"source": "const",
		"value":  1000,
	}

	for _, tc := range []struct {
		name       string
		maxacpower any
		expected   float64
	}{
		{"plugin", map[string]any{"source": "const", "value": 5000}, 5000},
		{"unavailable", map[string]any{"source": "error", "error": "ErrNotAvailable"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
				"power":      power,
				"maxacpower": tc.maxacpower,
			})
			require.NoError(t, err)

			mp, ok := api.Cap[api.MaxACPowerGetter](m)
			if tc.expected == 0 {
				assert.False(t, ok, "MaxACPowerGetter")
				return
			}

			require.True(t, ok, "MaxACPowerGetter")
			assert.Equal(t, tc.expected, mp.MaxACPower())
		})
	}
}
func TestBattery(t *testing.T) {
	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
		"capacity": 23,
		"soc": map[string]any{
			"source": "const",
			"value":  47,
		},
	})
	require.NoError(t, err)

	_, ok := api.Cap[api.Battery](m)
	assert.True(t, ok, "Battery")
	_, ok = api.Cap[api.BatteryCapacity](m)
	assert.True(t, ok, "BatteryCapacity")
}
