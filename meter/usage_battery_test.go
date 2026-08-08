package meter

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
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

// TestTemplateBatteryModes guards the declared batterymodes against the switch
// cases they mirror
func TestTemplateBatteryModes(t *testing.T) {
	caseModes := map[string]api.BatteryMode{
		"1": api.BatteryNormal,
		"2": api.BatteryHold,
		"3": api.BatteryCharge,
		"4": api.BatteryHoldCharge,
	}

	caseRE := regexp.MustCompile(`^\s*- case: (\d+)`)
	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }

	for _, tmpl := range templates.ByClass(templates.Meter) {
		lines := strings.Split(tmpl.Render, "\n")

		var declared []string
		var cases []api.BatteryMode

		for i, line := range lines {
			if _, after, ok := strings.Cut(line, "batterymodes: "); ok {
				require.NoError(t, yaml.Unmarshal([]byte(after), &declared), tmpl.Template)
			}

			if strings.TrimSpace(line) != "batterymode:" {
				continue
			}

			// scan until the next key at batterymode's own indentation
			for _, line := range lines[i+1:] {
				if strings.TrimSpace(line) != "" && indent(line) <= indent(lines[i]) {
					break
				}
				if m := caseRE.FindStringSubmatch(line); m != nil {
					// nested switches repeat cases, the supported set is their union
					if mode, ok := caseModes[m[1]]; ok && !slices.Contains(cases, mode) {
						cases = append(cases, mode)
					}
				}
			}
		}

		if len(cases) == 0 {
			continue // no switch-based batterymode
		}

		modes, err := batteryModes(declared)
		require.NoError(t, err, tmpl.Template)
		require.ElementsMatch(t, cases, modes, "%s: batterymodes must match the switch cases", tmpl.Template)
	}
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
