package server

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNaNInf(t *testing.T) {
	c := map[string]any{
		"foo": math.NaN(),
		"bar": math.Inf(0),
	}
	encodeFloats(c)
	assert.Equal(t, map[string]any{"foo": nil, "bar": nil}, c, "NaN not encoded as nil")
}
