package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/stretchr/testify/require"
)

func TestReverseConfigurable(t *testing.T) {
	t.Parallel()

	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  "123.4",
		},
		"currents": []plugin.Config{
			{Source: "const", Other: map[string]any{"value": "1"}},
			{Source: "const", Other: map[string]any{"value": "2"}},
			{Source: "const", Other: map[string]any{"value": "3"}},
		},
		"powers": []plugin.Config{
			{Source: "const", Other: map[string]any{"value": "10"}},
			{Source: "const", Other: map[string]any{"value": "20"}},
			{Source: "const", Other: map[string]any{"value": "30"}},
		},
	})
	require.NoError(t, err)

	m = Reverse(m)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, -123.4, p)

	c, ok := m.(api.PhaseCurrents)
	require.True(t, ok)
	l1, l2, l3, err := c.Currents()
	require.NoError(t, err)
	require.Equal(t, -1.0, l1)
	require.Equal(t, -2.0, l2)
	require.Equal(t, -3.0, l3)

	pp, ok := m.(api.PhasePowers)
	require.True(t, ok)
	l1, l2, l3, err = pp.Powers()
	require.NoError(t, err)
	require.Equal(t, -10.0, l1)
	require.Equal(t, -20.0, l2)
	require.Equal(t, -30.0, l3)
}

func TestReverseLeavesTotalEnergyUntouched(t *testing.T) {
	t.Parallel()

	m, err := NewConfigurableFromConfig(t.Context(), map[string]any{
		"power": map[string]any{
			"source": "const",
			"value":  "123.4",
		},
		"energy": map[string]any{
			"source": "const",
			"value":  "456.7",
		},
	})
	require.NoError(t, err)

	reversed := Reverse(m)

	me, ok := reversed.(api.MeterEnergy)
	require.True(t, ok)
	e, err := me.TotalEnergy()
	require.NoError(t, err)
	require.Equal(t, 456.7, e)
}

func TestNewFromTemplateConfigReversed(t *testing.T) {
	t.Parallel()

	m, err := NewFromTemplateConfig(t.Context(), map[string]any{
		"template":  "demo-meter",
		"usage":     "grid",
		"power":     123,
		"currentL1": 1,
		"currentL2": 2,
		"currentL3": 3,
		"reversed":  true,
	})
	require.NoError(t, err)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, -123.0, p)

	c, ok := m.(api.PhaseCurrents)
	require.True(t, ok)
	l1, l2, l3, err := c.Currents()
	require.NoError(t, err)
	require.Equal(t, -1.0, l1)
	require.Equal(t, -2.0, l2)
	require.Equal(t, -3.0, l3)
}

func TestNewFromConfigReversed(t *testing.T) {
	t.Parallel()

	m, err := NewFromConfig(t.Context(), api.Custom, map[string]any{
		"reversed": true,
		"power": map[string]any{
			"source": "const",
			"value":  "12",
		},
		"currents": []plugin.Config{
			{Source: "const", Other: map[string]any{"value": "1"}},
			{Source: "const", Other: map[string]any{"value": "2"}},
			{Source: "const", Other: map[string]any{"value": "3"}},
		},
		"energy": map[string]any{
			"source": "const",
			"value":  "9",
		},
	})
	require.NoError(t, err)

	p, err := m.CurrentPower()
	require.NoError(t, err)
	require.Equal(t, -12.0, p)

	c, ok := m.(api.PhaseCurrents)
	require.True(t, ok)
	l1, l2, l3, err := c.Currents()
	require.NoError(t, err)
	require.Equal(t, -1.0, l1)
	require.Equal(t, -2.0, l2)
	require.Equal(t, -3.0, l3)

	me, ok := m.(api.MeterEnergy)
	require.True(t, ok)
	e, err := me.TotalEnergy()
	require.NoError(t, err)
	require.Equal(t, 9.0, e)
}
