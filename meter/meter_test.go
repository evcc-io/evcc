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
	assert.Implements(t, new(api.MaxACPowerGetter), m)
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

	assert.Implements(t, new(api.Battery), m)
	assert.Implements(t, new(api.BatteryCapacity), m)
}
