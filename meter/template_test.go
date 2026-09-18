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

func TestFroniusGen24BatteryModes(t *testing.T) {
	tmpl, err := templates.ByName(templates.Meter, "fronius-gen24")
	require.NoError(t, err)

	b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeUnitTest, map[string]any{
		"host":  "localhost",
		"usage": "battery",
	})
	require.NoError(t, err)

	var conf struct {
		BatteryMode struct {
			Switch []struct {
				Case int64
				Set  struct {
					Set []struct {
						Value int64
						Set   struct {
							Value string
						}
					}
				}
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(b, &conf))

	keys := make([]int64, 0, len(conf.BatteryMode.Switch))
	for _, item := range conf.BatteryMode.Switch {
		keys = append(keys, item.Case)
	}

	require.Equal(t, []api.BatteryMode{
		api.BatteryNormal,
		api.BatteryHold,
		api.BatteryCharge,
		api.BatteryHoldCharge,
		api.BatteryDischarge,
	}, batteryModes(keys))

	var discharge int
	for i, item := range conf.BatteryMode.Switch {
		if item.Case == int64(api.BatteryDischarge) {
			discharge = i
			break
		}
	}

	writes := conf.BatteryMode.Switch[discharge].Set.Set
	require.Len(t, writes, 4)
	require.Equal(t, int64(0), writes[0].Value)
	require.Equal(t, "124:0:StorCtl_Mod", writes[0].Set.Value)
	require.Equal(t, int64(-100), writes[1].Value)
	require.Equal(t, "124:0:InWRte", writes[1].Set.Value)
	require.Equal(t, int64(100), writes[2].Value)
	require.Equal(t, "124:0:OutWRte", writes[2].Set.Value)
	require.Equal(t, int64(3), writes[3].Value)
	require.Equal(t, "124:0:StorCtl_Mod", writes[3].Set.Value)
}
