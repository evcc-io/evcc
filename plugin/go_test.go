package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A float parameter must keep its type inside the interpreter even when the
// value is a whole number. Otherwise arithmetic mixing it with a float constant
// fails with "invalid operation: mismatched types int and untyped float".
func TestGoFloatParam(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  int64
	}{
		{11000, 73},
		{15000, 100},
		{0, 0},
		{11000.5, 73},
	} {
		p, err := NewGoPluginFromConfig(t.Context(), map[string]any{
			"in": []map[string]any{{
				"name":   "limit",
				"type":   "float",
				"config": map[string]any{"source": "const", "value": tc.value},
			}},
			"script": "int(limit*100/15000 + 0.5)",
		})
		assert.NoError(t, err)

		g, err := p.(IntGetter).IntGetter()
		assert.NoError(t, err)

		v, err := g()
		assert.NoError(t, err)
		assert.Equal(t, tc.want, v, tc.value)
	}
}

// int parameters keep integer semantics
func TestGoIntParam(t *testing.T) {
	p, err := NewGoPluginFromConfig(t.Context(), map[string]any{
		"in": []map[string]any{{
			"name":   "mode",
			"type":   "int",
			"config": map[string]any{"source": "const", "value": 3},
		}},
		"script": "mode & 1",
	})
	assert.NoError(t, err)

	g, err := p.(IntGetter).IntGetter()
	assert.NoError(t, err)

	v, err := g()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)
}

// the value handed to a setter is a parameter, too
func TestGoFloatSetter(t *testing.T) {
	p, err := NewGoPluginFromConfig(t.Context(), map[string]any{
		"script": "int(limit*100/15000 + 0.5)",
	})
	assert.NoError(t, err)

	s, err := p.(FloatSetter).FloatSetter("limit")
	assert.NoError(t, err)
	assert.NoError(t, s(11000))
}
