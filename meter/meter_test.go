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
	assert.Implements(t, new(api.MaxACPowerGetter), m)
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

	assert.Implements(t, new(api.Battery), m)
	assert.Implements(t, new(api.BatteryCapacity), m)
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
		Meter:      base,
		curtailS:   func(v float64) error { limit = v; return nil },
		curtailedG: func() (float64, error) { return limit, nil },
	}

	// initially at 100 %, not curtailed
	curtailed, err := cm.Curtailed()
	require.NoError(t, err)
	assert.False(t, curtailed)

	// curtail → write 0
	require.NoError(t, cm.Curtail(true))
	assert.Equal(t, 0.0, limit)

	curtailed, err = cm.Curtailed()
	require.NoError(t, err)
	assert.True(t, curtailed)

	// restore → write 100
	require.NoError(t, cm.Curtail(false))
	assert.Equal(t, 100.0, limit)

	curtailed, err = cm.Curtailed()
	require.NoError(t, err)
	assert.False(t, curtailed)
}
