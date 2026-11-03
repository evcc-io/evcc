package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

// TestBatteryLimitSocWithBatteryMode covers limitsoc and batterymode configured together:
// the reserve soc is written first, then the mode
func TestBatteryLimitSocWithBatteryMode(t *testing.T) {
	ctx := context.TODO()

	js := func(script string) map[string]any {
		return map[string]any{"source": "js", "vm": "compose", "script": script}
	}

	m, err := NewConfigurableFromConfig(ctx, map[string]any{
		"power":        map[string]any{"source": "const", "value": 0},
		"soc":          map[string]any{"source": "const", "value": 42},
		"minsoc":       10,
		"maxsoc":       80,
		"limitsoc":     js(`log = (typeof log === "undefined" ? "" : log) + "L" + limitSoc + ";"`),
		"batterymode":  js(`log = (typeof log === "undefined" ? "" : log) + "M" + batteryMode + ";"`),
		"batterymodes": []string{"normal", "hold", "charge", "holdcharge"},
	})
	require.NoError(t, err)

	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge, api.BatteryHoldCharge}, ctrl.BatteryModes())

	for _, mode := range ctrl.BatteryModes() {
		require.NoError(t, ctrl.SetBatteryMode(mode))
	}

	var cc plugin.Config
	require.NoError(t, util.DecodeOther(js("log"), &cc))
	logG, err := cc.StringGetter(ctx)
	require.NoError(t, err)

	log, err := logG()
	require.NoError(t, err)

	// holdcharge is not expressible via limit soc, only the mode is written
	require.Equal(t, "L10;M1;L42;M2;L80;M3;M4;", log)
}
