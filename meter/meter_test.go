package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPV(t *testing.T) {
	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
		"maxacpower": 1000,
	})
	require.NoError(t, err)

	// must not have soc/capacity
	_, ok := api.Cap[api.MaxACPowerGetter](m)
	assert.True(t, ok, "MaxACPowerGetter")
}
func TestBattery(t *testing.T) {
	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
		"capacity": 23,
		"soc": map[string]any{
			"source": "const",
			"value":  47,
		},
	})
	require.NoError(t, err)

	_, ok := api.Cap[api.Battery](m)
	assert.True(t, ok, "Battery")
	_, ok = api.Cap[api.BatteryCapacity](m)
	assert.True(t, ok, "BatteryCapacity")
}

func TestCurtailPV(t *testing.T) {
	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  1000,
		},
		"curtailed": map[string]any{
			"source": "const",
			"value":  100,
		},
		"curtail": map[string]any{
			"source": "sleep",
		},
	})
	require.NoError(t, err)

	_, ok := m.(api.Curtailer)
	assert.True(t, ok, "meter should implement api.Curtailer")
}

func TestCurtailMeter(t *testing.T) {
	limit := 100.0

	base := &Meter{currentPowerG: func() (float64, error) { return 1000, nil }}
	cm := &curtailMeter{
		Meter:        base,
		curtailS:     func(v float64) error { limit = v; return nil },
		curtailedG:   func() (float64, error) { return limit, nil },
		nominalLimit: 100,
	}

	// initially at 100 %, not curtailed
	curtailed, err := cm.Curtailed()
	require.NoError(t, err)
	assert.False(t, curtailed)

	// curtail → write 0 (default curtailLimit)
	require.NoError(t, cm.Curtail(true))
	assert.Equal(t, 0.0, limit)

	curtailed, err = cm.Curtailed()
	require.NoError(t, err)
	assert.True(t, curtailed)

	// restore → write nominalLimit (100)
	require.NoError(t, cm.Curtail(false))
	assert.Equal(t, 100.0, limit)

	curtailed, err = cm.Curtailed()
	require.NoError(t, err)
	assert.False(t, curtailed)
}

// TestCurtailMeterFeedInLimit verifies that a site-specific feed-in cap (< 100 %)
// is used as the "uncurtailed" value and as the Curtailed() threshold.
func TestCurtailMeterFeedInLimit(t *testing.T) {
	limit := 60.0 // start at the legal feed-in cap

	base := &Meter{currentPowerG: func() (float64, error) { return 1000, nil }}
	cm := &curtailMeter{
		Meter:        base,
		curtailS:     func(v float64) error { limit = v; return nil },
		curtailedG:   func() (float64, error) { return limit, nil },
		curtailLimit: 0,
		nominalLimit: 60, // legal max feed-in limit for this installation
	}

	// at nominal limit → not curtailed
	curtailed, err := cm.Curtailed()
	require.NoError(t, err)
	assert.False(t, curtailed, "at nominalLimit (60%) should not be curtailed")

	// curtail → writes curtailLimit (0)
	require.NoError(t, cm.Curtail(true))
	assert.Equal(t, 0.0, limit)

	curtailed, err = cm.Curtailed()
	require.NoError(t, err)
	assert.True(t, curtailed, "at 0% should be curtailed")

	// restore → writes nominalLimit (60), NOT 100
	require.NoError(t, cm.Curtail(false))
	assert.Equal(t, 60.0, limit, "should restore to nominalLimit (60%), not 100%")

	curtailed, err = cm.Curtailed()
	require.NoError(t, err)
	assert.False(t, curtailed, "at nominalLimit (60%) should not be curtailed")
}
