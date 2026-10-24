package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMqttPresetBroker verifies the rendered broker. An empty host omits the
// broker so the evcc MQTT broker is used, a host with port or scheme is kept.
func TestMqttPresetBroker(t *testing.T) {
	tmpl, err := ByName(Charger, "tinkerforge-warp3")
	require.NoError(t, err)

	for _, tc := range []struct {
		host, broker string
	}{
		{"", ""},
		{"192.168.0.8", "192.168.0.8:1883"},
		{"192.168.0.8:1884", "192.168.0.8:1884"},
		{"tls://mqtt.example.com:8883", "tls://mqtt.example.com:8883"},
	} {
		t.Run(tc.host, func(t *testing.T) {
			b, _, err := tmpl.RenderResult(Charger, RenderModeInstance, map[string]any{"host": tc.host})
			require.NoError(t, err)

			if tc.broker == "" {
				assert.NotContains(t, string(b), "broker:")
			} else {
				assert.Contains(t, string(b), "\nbroker: "+tc.broker+"\n")
			}
		})
	}
}
