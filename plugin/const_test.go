package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConst(t *testing.T) {
	p, err := NewConstFromConfig(t.Context(), map[string]any{"value": nil})
	assert.NoError(t, err)

	{
		g, err := p.(StringGetter).StringGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, "", v)
	}

	{
		g, err := p.(IntGetter).IntGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), v)
	}

	{
		g, err := p.(FloatGetter).FloatGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, float64(0), v)
	}
}
