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

// TestSplitConfigPhasesHysteresis guards that the phase hysteresis keys
// (phasesuppower/phasesdownpower) are routed to the dynamic config, so they are
// applied live and never reach the strict static decode.
func TestSplitConfigPhasesHysteresis(t *testing.T) {
	payload := map[string]any{
		"phasesuppower":   4140.0,
		"phasesdownpower": 3000.0,
	}

	dynamic, other, err := SplitConfig(payload)
	require.NoError(t, err)

	require.NotNil(t, dynamic.PhasesUpPower)
	require.NotNil(t, dynamic.PhasesDownPower)
	assert.Equal(t, 4140.0, *dynamic.PhasesUpPower)
	assert.Equal(t, 3000.0, *dynamic.PhasesDownPower)
	assert.NotContains(t, other, "phasesuppower")
	assert.NotContains(t, other, "phasesdownpower")
}
