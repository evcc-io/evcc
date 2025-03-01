package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestACPower(t *testing.T) {
	m, err := NewConfigurableFromConfig(context.TODO(), map[string]any{
		"capacity":   23,
		"maxacpower": 1000,
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
	})
	require.NoError(t, err)
	_, ok := m.(api.BatteryCapacity)
	require.True(t, ok)
	_, ok = m.(api.MaxACPowerGetter)
	require.True(t, ok)
}
