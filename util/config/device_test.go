package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceRelease(t *testing.T) {
	h := &handler[int]{topic: "test"}

	var first, second int
	dev := NewConfigurableDevice(&Config{ID: 1, Type: "test"}, 1)
	dev.SetRelease(func() { first++ })
	require.NoError(t, h.Add(NewStaticDevice(Named{Name: "static"}, 0)))
	require.NoError(t, h.Add(dev))

	// replacing the release func releases the previous instance
	dev.SetRelease(func() { second++ })
	assert.Equal(t, 1, first)
	assert.Equal(t, 0, second)

	// deleting releases the current instance exactly once
	require.NoError(t, h.Delete(dev.Config().Name))
	assert.Equal(t, 1, second)

	dev.Release()
	assert.Equal(t, 1, first)
	assert.Equal(t, 1, second)

	// static devices have nothing to release
	require.NoError(t, h.Delete("static"))
}
