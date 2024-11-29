package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalc(t *testing.T) {
	p, err := NewCalcFromConfig(context.TODO(), map[string]any{
		"formula": "in0 * in1",
		"in": []map[string]any{
			{
				"source": "const",
				"value":  2,
			},
			{
				"source": "const",
				"value":  3,
			},
		},
	})
	require.NoError(t, err)

	fp := p.(FloatProvider)

	g, err := fp.FloatGetter()
	require.NoError(t, err)

	v, err := g()
	require.NoError(t, err)

	require.Equal(t, 6.0, v)
}
