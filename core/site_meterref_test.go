package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMeter struct {
	power float64
}

func (m testMeter) CurrentPower() (float64, error) {
	return m.power, nil
}

func TestMetersConfigDecodesMeterRefs(t *testing.T) {
	var cfg struct {
		Meters MetersConfig `mapstructure:"meters"`
	}

	err := util.DecodeOther(map[string]any{
		"meters": map[string]any{
			"battery": []any{
				"battery-1",
				map[string]any{
					"source": "battery-2",
					"title":  "Battery Upstairs",
				},
			},
			"ext": []any{
				map[string]any{
					"source": "ext-1",
					"title":  "Heat Pump",
				},
			},
		},
	}, &cfg)

	require.NoError(t, err)
	assert.Equal(t, MeterRefs{
		{Source: "battery-1"},
		{Source: "battery-2", Title: "Battery Upstairs"},
	}, cfg.Meters.BatteryMetersRef)
	assert.Equal(t, MeterRefs{
		{Source: "ext-1", Title: "Heat Pump"},
	}, cfg.Meters.ExtMetersRef)
}

func TestCollectMetersUsesRefTitleOverride(t *testing.T) {
	site := NewSite()

	meters := []config.Device[api.Meter]{
		config.NewStaticDevice(config.Named{Name: "battery-1"}, testMeter{power: 1200}),
	}
	refs := MeterRefs{
		{Source: "battery-1", Title: "Battery Basement"},
	}

	res := site.collectMeters("battery", meters, refs)
	require.Len(t, res, 1)
	assert.Equal(t, "Battery Basement", res[0].Title)
	assert.Equal(t, 1200.0, res[0].Power)
}
