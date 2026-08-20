package plugin

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestSwitchUnmatchedValue(t *testing.T) {
	cfg := map[string]any{
		"switch": []map[string]any{
			{"case": "1", "set": map[string]any{"source": "error", "error": "ErrAsleep"}},
		},
	}

	p, err := NewSwitchFromConfig(context.TODO(), cfg)
	require.NoError(t, err)

	set, err := p.(IntSetter).IntSetter("mode")
	require.NoError(t, err)

	// matching case is executed
	require.ErrorIs(t, set(1), api.ErrAsleep)

	// unmatched value without default is not available
	require.ErrorIs(t, set(3), api.ErrNotAvailable)

	// default takes precedence over the not available error
	cfg["default"] = map[string]any{"source": "error", "error": "ErrMustRetry"}

	p, err = NewSwitchFromConfig(context.TODO(), cfg)
	require.NoError(t, err)

	set, err = p.(IntSetter).IntSetter("mode")
	require.NoError(t, err)

	require.ErrorIs(t, set(3), api.ErrMustRetry)
}
