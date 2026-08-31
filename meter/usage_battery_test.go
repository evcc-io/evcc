package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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
	modes := func(t *testing.T, keys []int64, declared []string) []api.BatteryMode {
		t.Helper()
		res, err := batteryModes(keys, declared)
		require.NoError(t, err)
		return res
	}

	// a setter that doesn't switch on the mode and declares nothing has no mode withheld from it
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge, api.BatteryHoldCharge}, modes(t, nil, nil))

	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold}, modes(t, []int64{1, 2}, nil))

	// values that are not a mode are ignored, e.g. the marstek forced discharge case
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHoldCharge}, modes(t, []int64{1, 4, 5}, nil))

	require.Empty(t, modes(t, []int64{0}, nil))

	// the declaration fills in for a setter that cannot report its keys
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}, modes(t, nil, []string{"normal", "charge"}))

	// readable keys win over the declaration
	require.Equal(t, []api.BatteryMode{api.BatteryHold}, modes(t, []int64{2}, []string{"normal", "charge"}))

	_, err := batteryModes(nil, []string{"invalid"})
	require.Error(t, err)

	_, err = batteryModes(nil, []string{"unknown"})
	require.Error(t, err)
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
