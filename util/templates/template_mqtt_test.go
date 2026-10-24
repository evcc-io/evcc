package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMqttPresetDefaultBroker verifies that an empty mqtt host omits the broker
// so the evcc MQTT broker is used
func TestMqttPresetDefaultBroker(t *testing.T) {
	tmpl, err := ByName(Charger, "tinkerforge-warp3")
	require.NoError(t, err)

	b, _, err := tmpl.RenderResult(Charger, RenderModeInstance, map[string]any{"host": "192.168.0.8"})
	require.NoError(t, err)
	assert.Contains(t, string(b), "broker: 192.168.0.8:1883")

	b, _, err = tmpl.RenderResult(Charger, RenderModeInstance, nil)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "broker:")
}
