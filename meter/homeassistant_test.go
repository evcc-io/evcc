package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// TestHomeAssistantBatteryModes covers which modes a partial mode config announces
func TestHomeAssistantBatteryModes(t *testing.T) {
	conf := func(hold, charge string) map[string]any {
		return map[string]any{
			"uri":        "http://localhost",
			"token":      "foo",
			"power":      "sensor.power",
			"soc":        "sensor.soc",
			"modeNormal": "script.normal",
			"modeHold":   hold,
			"modeCharge": charge,
		}
	}

	ctx := context.Background()
	m, err := NewHomeAssistantFromConfig(ctx, conf("script.hold", "script.charge"))
	require.NoError(t, err)

	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge}, ctrl.BatteryModes())

	// a mode without entity is not announced and rejected by the setter
	m, err = NewHomeAssistantFromConfig(ctx, conf("script.hold", ""))
	require.NoError(t, err)

	ctrl, ok = api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold}, ctrl.BatteryModes())
	require.Error(t, ctrl.SetBatteryMode(api.BatteryCharge))

	// a mode entity must be a script
	_, err = NewHomeAssistantFromConfig(ctx, conf("switch.hold", ""))
	require.Error(t, err)

	// holdcharge and discharge are announced when configured
	c := conf("", "")
	c["modeHoldCharge"] = "script.holdcharge"
	c["modeDischarge"] = "script.discharge"
	m, err = NewHomeAssistantFromConfig(c)
	require.NoError(t, err)

	ctrl, ok = api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHoldCharge, api.BatteryDischarge}, ctrl.BatteryModes())

	// modeNormal is required with any other mode
	c = conf("", "")
	c["modeNormal"] = ""
	c["modeDischarge"] = "script.discharge"
	_, err = NewHomeAssistantFromConfig(c)
	require.Error(t, err)

	// modeNormal alone is rejected
	_, err = NewHomeAssistantFromConfig(conf("", ""))
	require.Error(t, err)
}

// TestHomeAssistantTemplateSocLimits covers rendering of static and entity-based soc limits
func TestHomeAssistantTemplateSocLimits(t *testing.T) {
	tmpl, err := templates.ByName(templates.Meter, "homeassistant")
	require.NoError(t, err)

	render := func(extra map[string]any) map[string]any {
		values := map[string]any{
			"usage": "battery",
			"uri":   "http://localhost:8123",
			"power": "sensor.power",
			"soc":   "sensor.soc",
		}
		for k, v := range extra {
			values[k] = v
		}

		b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, values)
		require.NoError(t, err)

		var res map[string]any
		require.NoError(t, yaml.Unmarshal(b, &res), string(b))
		return res
	}

	res := render(map[string]any{
		"minSocEntity": "number.min_soc",
		"maxSocEntity": "number.max_soc",
		"maxsoc":       90,
	})
	require.Equal(t, "homeassistant", res["minsoc"].(map[string]any)["source"])
	require.Equal(t, "number.min_soc", res["minsoc"].(map[string]any)["entity"])
	require.Equal(t, "number.max_soc", res["maxsoc"].(map[string]any)["entity"], "entity overrides static value")

	res = render(map[string]any{"minsoc": 10, "maxsoc": 90})
	require.EqualValues(t, 10, res["minsoc"])
	require.EqualValues(t, 90, res["maxsoc"])
}
