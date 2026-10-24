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

func TestBatteryModePower(t *testing.T) {
	ctx := context.TODO()
	js := func(vm, param string) map[string]any {
		return map[string]any{
			"source": "js",
			"vm":     vm,
			"script": param,
		}
	}

	m, err := NewConfigurableFromConfig(ctx, map[string]any{
		"power": map[string]any{"source": "const", "value": 0},
		"soc":   map[string]any{"source": "const", "value": 42},
		"batterymodepower": map[string]any{
			"charge":    js("mode-power-charge-test", "batteryChargePower"),
			"discharge": js("mode-power-discharge-test", "batteryDischargePower"),
		},
	})
	require.NoError(t, err)

	ctrl, ok := api.Cap[api.BatteryModePowerController](m)
	require.True(t, ok)
	require.NoError(t, ctrl.SetBatteryModePower(api.BatteryCharge, 1234))
	require.NoError(t, ctrl.SetBatteryModePower(api.BatteryDischarge, 567))

	var chargeConfig, dischargeConfig plugin.Config
	require.NoError(t, util.DecodeOther(js("mode-power-charge-test", "batteryChargePower"), &chargeConfig))
	require.NoError(t, util.DecodeOther(js("mode-power-discharge-test", "batteryDischargePower"), &dischargeConfig))
	chargeG, err := chargeConfig.FloatGetter(ctx)
	require.NoError(t, err)
	dischargeG, err := dischargeConfig.FloatGetter(ctx)
	require.NoError(t, err)
	charge, err := chargeG()
	require.NoError(t, err)
	discharge, err := dischargeG()
	require.NoError(t, err)
	require.Equal(t, float64(1234), charge)
	require.Equal(t, float64(567), discharge)
}

func TestNewFromConfigKeepsMasterOutOfDriverConfig(t *testing.T) {
	other := map[string]any{
		"master": "db:2",
		"power":  map[string]any{"source": "const", "value": 0},
	}

	_, err := NewFromConfig(context.TODO(), api.Custom, other)
	require.NoError(t, err)
	require.Equal(t, "db:2", other["master"])
}
