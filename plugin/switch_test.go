package plugin

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestSwitchUnmatchedValue(t *testing.T) {
	// error plugins make the executed branch observable
	cases := []map[string]any{
		{"case": "1", "set": map[string]any{"source": "error", "error": "ErrAsleep"}},
	}

	p, err := NewSwitchFromConfig(t.Context(), map[string]any{"switch": cases})
	require.NoError(t, err)

	set, err := p.(IntSetter).IntSetter("mode")
	require.NoError(t, err)

	require.ErrorIs(t, set(1), api.ErrAsleep)
	require.ErrorIs(t, set(3), api.ErrNotAvailable)

	// explicit default wins over the not available error
	p, err = NewSwitchFromConfig(t.Context(), map[string]any{
		"switch":  cases,
		"default": map[string]any{"source": "error", "error": "ErrMustRetry"},
	})
	require.NoError(t, err)

	set, err = p.(IntSetter).IntSetter("mode")
	require.NoError(t, err)

	require.ErrorIs(t, set(3), api.ErrMustRetry)
}
