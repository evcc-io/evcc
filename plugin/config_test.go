package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredConfig(t *testing.T) {
	var c Config

	_, err := c.BoolGetter(t.Context())
	assert.Error(t, err)

	_, err = c.IntSetter(t.Context(), "foo")
	assert.Error(t, err)

	c = Config{
		Source: "http",
		Other:  map[string]any{"uri": "http://localhost"},
	}

	g, err := c.BoolGetter(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, g)

	s, err := c.IntSetter(t.Context(), "foo")
	assert.NoError(t, err)
	assert.NotNil(t, s)

	c = Config{Source: "foo"}

	_, err = c.BoolGetter(t.Context())
	assert.Error(t, err)
}

func TestOptionalConfig(t *testing.T) {
	var c *Config

	g, err := c.BoolGetter(t.Context())
	assert.NoError(t, err)
	assert.Nil(t, g)

	s, err := c.IntSetter(t.Context(), "foo")
	assert.NoError(t, err)
	assert.Nil(t, s)
}
