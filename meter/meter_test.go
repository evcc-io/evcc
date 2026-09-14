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
	newMeter := func(maxacpower any) api.Meter {
		m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
			"power":      map[string]any{"source": "const", "value": 1000},
			"maxacpower": maxacpower,
		})
		require.NoError(t, err)
		return m
	}

	m := newMeter(map[string]any{"source": "const", "value": 5000})
	mp, ok := api.Cap[api.MaxACPowerGetter](m)
	require.True(t, ok, "MaxACPowerGetter")
	assert.Equal(t, 5000.0, mp.MaxACPower())

	// device without rating must not be decorated
	m = newMeter(map[string]any{"source": "error", "error": "ErrNotAvailable"})
	_, ok = api.Cap[api.MaxACPowerGetter](m)
	assert.False(t, ok, "MaxACPowerGetter")
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
