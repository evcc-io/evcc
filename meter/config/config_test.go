package config

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMeter struct {
	implement.Caps
}

func (m *testMeter) CurrentPower() (float64, error) {
	return 0, nil
}

func TestEfficiency(t *testing.T) {
	var received map[string]any

	Registry.Add("test-efficiency", func(other map[string]any) (api.Meter, error) {
		received = other
		return &testMeter{Caps: implement.New()}, nil
	})

	for _, tc := range []struct {
		val      any
		expected float64
	}{
		{nil, 0},
		{"", 0},
		{95, 95},
		{"95", 95},
		{95.5, 95.5},
	} {
		m, err := NewFromConfig(context.TODO(), "test-efficiency", map[string]any{"efficiency": tc.val, "foo": "bar"})
		require.NoError(t, err, tc)

		// efficiency is not passed on to the device
		assert.NotContains(t, received, "efficiency", tc)
		assert.Contains(t, received, "foo", tc)

		eff, ok := api.Cap[api.BatteryEfficiency](m)
		if tc.expected == 0 {
			assert.False(t, ok, tc)
			continue
		}

		require.True(t, ok, tc)
		assert.Equal(t, tc.expected, eff.GetEfficiency(), tc)
	}

	for _, val := range []any{0, -1, 101, "foo"} {
		_, err := NewFromConfig(context.TODO(), "test-efficiency", map[string]any{"efficiency": val})
		assert.Error(t, err, val)
	}
}
