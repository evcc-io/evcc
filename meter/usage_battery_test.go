package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/require"
)

func TestBatteryCapacity(t *testing.T) {
	ctx := context.TODO()

	// static value
	{
		var cc batteryCapacityCtx
		require.NoError(t, util.DecodeOther(map[string]any{"capacity": 10}, &cc))
		g, err := cc.Decorator(ctx)
		require.NoError(t, err)
		require.NotNil(t, g)
		require.Equal(t, 10.0, g())
	}

	// zero value is treated as not configured
	{
		var cc batteryCapacityCtx
		require.NoError(t, util.DecodeOther(map[string]any{"capacity": 0}, &cc))
		g, err := cc.Decorator(ctx)
		require.NoError(t, err)
		require.Nil(t, g)
	}

	// unset is not configured
	{
		var cc batteryCapacityCtx
		g, err := cc.Decorator(ctx)
		require.NoError(t, err)
		require.Nil(t, g)
	}

	// float plugin
	{
		var cc batteryCapacityCtx
		require.NoError(t, util.DecodeOther(map[string]any{
			"capacity": map[string]any{
				"source": "const",
				"value":  "12.5",
			},
		}, &cc))
		g, err := cc.Decorator(ctx)
		require.NoError(t, err)
		require.NotNil(t, g)
		require.Equal(t, 12.5, g())
	}
}

func TestBatteryModes(t *testing.T) {
	// unset defaults to the modes evcc assumed before batterymodes existed
	modes, err := batteryModes(nil)
	require.NoError(t, err)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge}, modes)

	modes, err = batteryModes([]string{"normal", " hold ", "holdcharge"})
	require.NoError(t, err)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryHoldCharge}, modes)

	_, err = batteryModes([]string{"sell"})
	require.Error(t, err)

	_, err = batteryModes([]string{"unknown"})
	require.Error(t, err)
}

// switchCases collects the switch case values below v. Nested switches repeat
// cases, so the result is their union.
func switchCases(v any, res map[int]struct{}) {
	switch v := v.(type) {
	case map[string]any:
		for key, val := range v {
			if key == "switch" {
				if cases, ok := val.([]any); ok {
					for _, c := range cases {
						if c, ok := c.(map[string]any); ok {
							if i, err := cast.ToIntE(c["case"]); err == nil {
								res[i] = struct{}{}
							}
						}
					}
				}
			}
			switchCases(val, res)
		}
	case []any:
		for _, val := range v {
			switchCases(val, res)
		}
	}
}

// TestTemplateBatteryModes guards the declared batterymodes against the switch
// cases they mirror
func TestTemplateBatteryModes(t *testing.T) {
	caseModes := map[int]api.BatteryMode{
		1: api.BatteryNormal,
		2: api.BatteryHold,
		3: api.BatteryCharge,
		4: api.BatteryHoldCharge,
	}

	templates.TestClass(t, templates.Meter, func(t *testing.T, values map[string]any) {
		t.Helper()

		instance, err := templates.RenderInstance(templates.Meter, values)
		if err != nil {
			return // covered by TestTemplates
		}

		cases := make(map[int]struct{})
		switchCases(instance.Other["batterymode"], cases)
		if len(cases) == 0 {
			return // no switch-based batterymode
		}

		expected := make([]api.BatteryMode, 0, len(cases))
		for c := range cases {
			mode, ok := caseModes[c]
			require.True(t, ok, "unmapped battery mode case: %d", c)
			expected = append(expected, mode)
		}

		var declared []string
		require.NoError(t, util.DecodeOther(instance.Other["batterymodes"], &declared))

		modes, err := batteryModes(declared)
		require.NoError(t, err)
		require.ElementsMatch(t, expected, modes, "batterymodes must match the switch cases")
	})
}

func TestBatterySocLimits(t *testing.T) {
	other := map[string]any{
		"minsoc": 1,
		"maxsoc": 99,
	}

	expected := batterySocLimits{
		MinSoc: 1,
		MaxSoc: 99,
	}

	{
		var res batterySocLimits
		require.NoError(t, util.DecodeOther(other, &res))
		require.Equal(t, expected, res)
	}

	{
		var res struct {
			batterySocLimits `mapstructure:",squash"`
		}
		require.NoError(t, util.DecodeOther(other, &res))
		require.Equal(t, expected, res.batterySocLimits)
	}

	{
		var res struct {
			BatterySocLimits batterySocLimits `mapstructure:",squash"`
		}
		require.NoError(t, util.DecodeOther(other, &res))
		require.Equal(t, expected, res.BatterySocLimits)
	}

	{
		res := struct {
			batterySocLimits `mapstructure:",squash"`
		}{
			batterySocLimits: batterySocLimits{
				MinSoc: 20,
				MaxSoc: 95,
			},
		}
		require.NoError(t, util.DecodeOther(other, &res))
		require.Equal(t, expected, res.batterySocLimits)
	}

	{
		res := struct {
			pvMaxACPower     `mapstructure:",squash"`
			batterySocLimits `mapstructure:",squash"`
		}{
			batterySocLimits: batterySocLimits{
				MinSoc: 20,
				MaxSoc: 95,
			},
		}
		require.NoError(t, util.DecodeOther(other, &res))
		require.Equal(t, expected, res.batterySocLimits)
	}
}
