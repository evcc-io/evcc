package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var c *Config

	g, err := c.BoolGetter(context.TODO())
	assert.NoError(t, err)
	assert.Nil(t, g)

	s, err := c.IntSetter(context.TODO(), "foo")
	assert.NoError(t, err)
	assert.Nil(t, s)
}
