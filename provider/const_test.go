package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConst(t *testing.T) {
	p, err := NewConstFromConfig(context.TODO(), map[string]interface{}{"value": nil})
	assert.NoError(t, err)

	{
		g, err := p.(StringProvider).StringGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, "", v)
	}

	{
		g, err := p.(IntProvider).IntGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), v)
	}

	{
		g, err := p.(FloatProvider).FloatGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, float64(0), v)
	}
}
