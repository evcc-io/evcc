package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstance(t *testing.T) {
	// device instances are interfaces, a disabled device holds a nil interface
	h := &handler[any]{topic: "test"}
	one := 1
	require.NoError(t, h.Add(NewStaticDevice[any](Named{Name: "one"}, &one)))
	require.NoError(t, h.Add(NewStaticDevice[any](Named{Name: "disabled"}, nil)))

	res, err := Instance(h, "one")
	require.NoError(t, err)
	assert.Same(t, &one, res)

	_, err = Instance(h, "disabled")
	require.Error(t, err, "disabled device has no instance")

	_, err = Instance(h, "missing")
	require.ErrorIs(t, err, ErrNotFound)
}
