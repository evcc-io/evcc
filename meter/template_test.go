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

func TestSolaxX3IES(t *testing.T) {
	rendered, values := renderSolaxBattery(t, "X3-IES")
	require.Contains(t, string(rendered), "address: 22 # 0x0016 Batpower_Charge1")
	require.Contains(t, string(rendered), "address: 28 # 0x001C Battery 1 Capacity")
	require.Contains(t, string(rendered), "address: 147 # 0x0093 Self-use discharge minimum SoC in high byte")
	require.Contains(t, string(rendered), "float64(value >> 8)")
	require.Contains(t, string(rendered), "address: 270 # 0x010E Battery charge upper SoC readback")
	require.Contains(t, string(rendered), "address: 38 # 0x0026 Battery 1 total energy (nominal capacity)")
	require.Contains(t, string(rendered), "maxchargepower:")

	device, err := NewFromConfig(t.Context(), "template", values)
	require.NoError(t, err)
	_, ok := api.Cap[api.BatteryCapacity](device)
	require.True(t, ok)
	_, ok = api.Cap[api.BatterySocLimiter](device)
	require.True(t, ok)
}

func TestSolaxG3G4(t *testing.T) {
	rendered, _ := renderSolaxBattery(t, "G3/G4", map[string]any{
		"capacity": "13.5",
		"minsoc":   "24",
		"maxsoc":   "91",
	})

	require.Contains(t, string(rendered), "capacity: 13.5 # kWh")
	require.Contains(t, string(rendered), "minsoc: 24 # %")
	require.Contains(t, string(rendered), "maxsoc: 91 # %")
	require.NotContains(t, string(rendered), "Self-use discharge minimum SoC in high byte")
	require.NotContains(t, string(rendered), "Battery charge upper SoC readback")
}

func renderSolaxBattery(t *testing.T, model string, overrides ...map[string]any) ([]byte, map[string]any) {
	t.Helper()

	tmpl, err := templates.ByName(templates.Meter, "solax")
	require.NoError(t, err)

	values := map[string]any{
		"template": "solax",
		"usage":    "battery",
		"model":    model,
		"modbus":   templates.ModbusChoiceTCPIP,
		"host":     "localhost",
	}
	for _, override := range overrides {
		for key, value := range override {
			values[key] = value
		}
	}

	rendered, values, err := tmpl.RenderResult(templates.Meter, templates.RenderModeUnitTest, values)
	require.NoError(t, err)

	var config map[string]any
	require.NoError(t, yaml.Unmarshal(rendered, &config))

	return rendered, values
}
