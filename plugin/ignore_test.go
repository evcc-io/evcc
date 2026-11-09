package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoreFloatGetter(t *testing.T) {
	for _, tc := range []struct {
		match string
		err   bool
	}{
		{"exit status 1", false},
		{"other", true},
	} {
		p, err := NewIgnoreFromConfig(t.Context(), map[string]any{
			"error": tc.match,
			"get":   map[string]any{"source": "script", "cmd": "false"},
		})
		require.NoError(t, err)

		g, err := p.(FloatGetter).FloatGetter()
		require.NoError(t, err)

		v, err := g()
		assert.Equal(t, tc.err, err != nil, tc.match)
		assert.Equal(t, 0.0, v)
	}
}
