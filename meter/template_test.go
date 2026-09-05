package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

var acceptable = []string{
	api.ErrMissingCredentials.Error(),
	api.ErrMissingToken.Error(),
	"invalid plugin source: ...",
	"missing mqtt broker configuration",
	"mqtt not configured",
	"not a SunSpec device",
	"connect: connection refused", // sockets
	"power: timeout",              // sockets
	"missing password",            // Powerwall
	"connect: no route to host",
	"connect: connection refused",
	"connect: network is unreachable",
	"i/o timeout",
	"timeout",                      // RCT
	"'sma': missing uri or serial", // SMA
	"[1ESY1161052714 1ESY1161229249 1EMH0008842285 1ESY1161978584 1EMH0004864048 1ESY1161979033 7ELS8135823805]", // Discovergy
	"can only have either uri or device",                                   // modbus
	"connection already registered with different protocol: localhost:502", // modbus
	"(Client.Timeout exceeded while awaiting headers)",                     // http
	"context deadline exceeded",                                            // LG ESS
	"no ping response for 192.0.2.2",                                       // SMA
	"no Speedwire ping response for 127.0.0.1",                             // SMA
	"no such network interface",                                            // SMA
	"missing config values: username, password, key",                       // E3DC
	"missing access key",                                                   // Ecoflow
	"eebus not configured",                                                 // EEBus
	"missing token",                                                        // HomeAssistant
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Meter, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig(t.Context(), "template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}

func TestAtmoceBatteryModes(t *testing.T) {
	for _, firmware := range []string{"<01.01.00.28.15", ">=01.01.00.28.15"} {
		t.Run(firmware, func(t *testing.T) {
			tmpl, err := templates.ByName(templates.Meter, "atmoce")
			require.NoError(t, err)
			data, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, map[string]any{
				"usage": "battery", "firmware": firmware, "modbus": "tcpip", "host": "localhost", "port": 15020,
			})
			require.NoError(t, err)
			var conf map[string]any
			require.NoError(t, yaml.Unmarshal(data, &conf))
			require.NotContains(t, conf, "batterymodes")
			delete(conf, "type")

			m, err := NewConfigurableFromConfig(t.Context(), conf)
			require.NoError(t, err)
			ctrl, ok := api.Cap[api.BatteryController](m)
			require.True(t, ok)
			require.Equal(t, []api.BatteryMode{
				api.BatteryNormal, api.BatteryHold, api.BatteryCharge, api.BatteryHoldCharge, api.BatteryDischarge,
			}, ctrl.BatteryModes())
		})
	}
}
