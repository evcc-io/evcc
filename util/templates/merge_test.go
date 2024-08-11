package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeMaps(t *testing.T) {
	target := map[string]any{
		"foo": "bar",
		"nested": map[string]any{
			"bar": "baz",
		},
	}
	other := map[string]any{
		"Foo": 1,
		"Nested": map[string]any{
			"Bar": 2,
		},
	}

	require.NoError(t, mergeMaps(other, target))
	require.Equal(t, map[string]any{
		"foo": 1,
		"nested": map[string]any{
			"bar": 2,
		},
	}, target)
}
