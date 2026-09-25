package measurement

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func constConfig(value string) *plugin.Config {
	return &plugin.Config{
		Source: "const",
		Other: map[string]any{
			"value": value,
		},
	}
}

func TestTemperatureConfigureHeatingAndWater(t *testing.T) {
	ctx := context.Background()

	cc := Temperature{
		Temp:             constConfig("42"),
		LimitTemp:        constConfig("84"),
		TempHeating:      constConfig("35"),
		LimitTempHeating: constConfig("55"),
		TempWater:        constConfig("48"),
		LimitTempWater:   constConfig("60"),
	}

	tempG, limitTempG, err := cc.Configure(ctx)
	require.NoError(t, err)
	val, err := tempG()
	require.NoError(t, err)
	assert.Equal(t, 42.0, val)
	limitVal, err := limitTempG()
	require.NoError(t, err)
	assert.Equal(t, int64(84), limitVal)

	tempHeatingG, limitTempHeatingG, err := cc.ConfigureHeating(ctx)
	require.NoError(t, err)
	val, err = tempHeatingG()
	require.NoError(t, err)
	assert.Equal(t, 35.0, val)
	limitVal, err = limitTempHeatingG()
	require.NoError(t, err)
	assert.Equal(t, int64(55), limitVal)

	tempWaterG, limitTempWaterG, err := cc.ConfigureWater(ctx)
	require.NoError(t, err)
	val, err = tempWaterG()
	require.NoError(t, err)
	assert.Equal(t, 48.0, val)
	limitVal, err = limitTempWaterG()
	require.NoError(t, err)
	assert.Equal(t, int64(60), limitVal)
}

func TestTemperatureConfigureHeatingAndWaterOptional(t *testing.T) {
	ctx := context.Background()

	var cc Temperature

	tempHeatingG, limitTempHeatingG, err := cc.ConfigureHeating(ctx)
	require.NoError(t, err)
	assert.Nil(t, tempHeatingG)
	assert.Nil(t, limitTempHeatingG)

	tempWaterG, limitTempWaterG, err := cc.ConfigureWater(ctx)
	require.NoError(t, err)
	assert.Nil(t, tempWaterG)
	assert.Nil(t, limitTempWaterG)
}
