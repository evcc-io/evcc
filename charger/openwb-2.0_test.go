package charger

import (
	"bytes"
	"context"
	"testing"

	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestOpenWB20DisplayConfig(t *testing.T) {
	require.Zero(t, network.Config().Port)
	log := util.NewLogger("openwb-2.0")
	writer := log.DEBUG.Writer()
	t.Cleanup(func() { log.DEBUG.SetOutput(writer) })

	for _, test := range []struct {
		name    string
		display any
		message string
	}{
		{"default", nil, "display setup skipped: network not initialized"},
		{"enabled", true, "display setup skipped: network not initialized"},
		{"disabled", false, "display setup disabled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			log.DEBUG.SetOutput(&output)
			config := map[string]any{"uri": "localhost:1502"}
			if test.display != nil {
				config["display"] = test.display
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := NewOpenWB20FromConfig(ctx, config)
			require.NoError(t, err)
			assert.Contains(t, output.String(), test.message)
		})
	}
}

func TestOpenWB20DisplayTemplate(t *testing.T) {
	for _, tmpl := range templates.ByClass(templates.Charger) {
		if tmpl.Template != "openwb-2.0" {
			continue
		}
		for _, enabled := range []bool{true, false} {
			values := tmpl.Defaults(templates.RenderModeUnitTest)
			assert.Equal(t, "true", values["display"])
			values["host"] = "localhost"
			values["display"] = enabled
			values["broker"] = "tls://openwb:8883"
			values["user"] = "display"
			values["password"] = "test-password"
			values[templates.ModbusKeyTCPIP] = true
			tmpl.ModbusValues(templates.RenderModeUnitTest, values)
			data, _, err := tmpl.RenderResult(templates.Charger, templates.RenderModeInstance, values)
			require.NoError(t, err)
			var config map[string]any
			require.NoError(t, yaml.Unmarshal(data, &config))
			assert.Equal(t, enabled, config["display"])
			if enabled {
				assert.Equal(t, values["broker"], config["broker"])
				assert.Equal(t, values["user"], config["user"])
				assert.Equal(t, values["password"], config["password"])
			} else {
				assert.NotContains(t, config, "broker")
			}
		}
		return
	}
	t.Fatal("openwb-2.0 template not found")
}
