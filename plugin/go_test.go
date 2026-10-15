package plugin

import (
	"reflect"
	"testing"

	"github.com/evcc-io/evcc/plugin/golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/yaegi/interp"
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

// the script is compiled once and sees the current inputs on each invocation without growing the VM
func TestGoCompileOnce(t *testing.T) {
	var i int64
	p := &Go{
		vm:     &golang.VM{Interpreter: interp.New(interp.Options{})},
		script: "res := mode * 2\nres",
		in:     []inputTransformation{{name: "mode", function: func() (any, error) { i++; return i, nil }}},
		prg:    make(map[string]*interp.Program),
	}

	roots := func() int {
		return reflect.ValueOf(p.vm.Interpreter).Elem().FieldByName("roots").Len()
	}

	var n int
	for j := range int64(100) {
		v, err := p.handleGetter()
		require.NoError(t, err)
		assert.Equal(t, 2*(j+1), v)

		if j == 0 {
			n = roots()
		}
	}

	assert.Equal(t, n, roots(), "interpreter must not grow on invocation")
}

// plugins sharing a named VM share parameters of the same name and type
func TestGoSharedVM(t *testing.T) {
	plugin := func(script string) (*Go, error) {
		p, err := NewGoPluginFromConfig(t.Context(), map[string]any{
			"vm":     t.Name(),
			"script": script,
		})
		if err != nil {
			return nil, err
		}
		return p.(*Go), nil
	}

	p1, err := plugin("x + 1")
	require.NoError(t, err)
	p2, err := plugin("x * 10")
	require.NoError(t, err)

	v, err := p1.evaluate("x", []goParam{{"x", int64(1)}})
	require.NoError(t, err)
	assert.Equal(t, int64(2), v)

	v, err = p2.evaluate("x", []goParam{{"x", int64(2)}})
	require.NoError(t, err)
	assert.Equal(t, int64(20), v)

	v, err = p1.evaluate("x", []goParam{{"x", int64(3)}})
	require.NoError(t, err)
	assert.Equal(t, int64(4), v)

	_, err = p2.evaluate("y", []goParam{{"x", "foo"}})
	assert.ErrorContains(t, err, "conflicts")
}

// runtime panics of the script are returned as error
func TestGoPanic(t *testing.T) {
	p, err := NewGoPluginFromConfig(t.Context(), map[string]any{
		"script": "[]int{1}[idx]",
	})
	require.NoError(t, err)

	_, err = p.(*Go).evaluate("idx", []goParam{{"idx", int64(5)}})
	assert.ErrorContains(t, err, "index out of range")
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
