package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold}, batteryModes([]int64{1, 2}))

	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHoldCharge, api.BatteryDischarge}, batteryModes([]int64{1, 4, 5}))

	// values that are not a mode are ignored
	require.Equal(t, []api.BatteryMode{api.BatteryNormal}, batteryModes([]int64{1, 99}))

	require.Empty(t, batteryModes([]int64{0}))
}

func TestDeclaredBatteryModes(t *testing.T) {
	modes, err := declaredBatteryModes([]string{"normal", "charge"})
	require.NoError(t, err)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}, modes)

	// nothing declared is nothing supported, the caller rejects it
	modes, err = declaredBatteryModes(nil)
	require.NoError(t, err)
	require.Empty(t, modes)

	_, err = declaredBatteryModes([]string{"invalid"})
	require.Error(t, err)

	// unknown parses as an enum value but is not a mode
	_, err = declaredBatteryModes([]string{"unknown"})
	require.Error(t, err)
}

// TestConfigurableBatteryModes covers where a configurable meter takes its modes from
func TestConfigurableBatteryModes(t *testing.T) {
	// a switch reports its cases, a default or a wrapping sequence hides them
	const readable = `
  source: switch
  switch:
  - case: 1
    set: {source: sleep, duration: 0s}
  - case: 2
    set: {source: sleep, duration: 0s}`

	const unreadable = `
  source: sequence
  set:
  - source: switch
    switch:
    - case: 1
      set: {source: sleep, duration: 0s}`

	const withDefault = readable + `
  default: {source: sleep, duration: 0s}`

	for _, tc := range []struct {
		name     string
		setter   string
		declared string
		modes    []api.BatteryMode
	}{
		{"keys", readable, "", []api.BatteryMode{api.BatteryNormal, api.BatteryHold}},
		{"keys win over declaration", readable, `["charge"]`, []api.BatteryMode{api.BatteryNormal, api.BatteryHold}},
		{"declaration for a hidden switch", unreadable, `["normal", "charge"]`, []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}},
		{"declaration for a default", withDefault, `["normal"]`, []api.BatteryMode{api.BatteryNormal}},
		{"neither keys nor declaration", unreadable, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := "power: {source: const, value: 0}\nsoc: {source: const, value: 50}\nbatterymode:" + tc.setter + "\n"
			if tc.declared != "" {
				conf += "batterymodes: " + tc.declared + "\n"
			}

			var other map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(conf), &other))

			m, err := NewConfigurableFromConfig(context.TODO(), other)
			if tc.modes == nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			ctrl, ok := api.Cap[api.BatteryController](m)
			require.True(t, ok)
			require.Equal(t, tc.modes, ctrl.BatteryModes())
		})
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
