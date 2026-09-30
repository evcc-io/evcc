package loadpoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitConfigUI(t *testing.T) {
	payload := map[string]any{
		"title": "Water Heater",
		"ui": map[string]any{
			"minTemp": 20.0,
			"maxTemp": 45.0,
		},
	}

	dynamic, other, err := SplitConfig(payload)
	require.NoError(t, err)

	assert.Equal(t, 20.0, dynamic.UI.MinTemp)
	assert.Equal(t, 45.0, dynamic.UI.MaxTemp)
	assert.NotContains(t, other, "ui")
}

// TestSplitConfigAlwaysCharge guards that alwaysCharge is treated as dynamic
// config. It is persisted per loadpoint (and, for database-configured loadpoints,
// written back into the config blob by the pv/minpv self-heal migration), so it
// must not reach the strict static decode in NewLoadpointFromConfig.
func TestSplitConfigAlwaysCharge(t *testing.T) {
	payload := map[string]any{
		"charger":      "wallbox",
		"alwaysCharge": "on",
	}

	dynamic, other, err := SplitConfig(payload)
	require.NoError(t, err)

	assert.Equal(t, "on", dynamic.AlwaysCharge)
	assert.NotContains(t, other, "alwaysCharge")
}
