package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACPower(t *testing.T) {
	m, err := NewConfigurableFromConfig(context.TODO(), map[string]any{
		"capacity": 23,
		"soc": map[string]any{
			"source": "const",
			"value":  47,
		},
		"maxacpower": 1000,
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
	})
	require.NoError(t, err)
	_, ok := m.(api.BatteryCapacity)
	assert.True(t, ok, "api.BatteryCapacity")
	_, ok = m.(api.MaxACPowerGetter)
	assert.True(t, ok, "api.MaxACPowerGetter")
}
