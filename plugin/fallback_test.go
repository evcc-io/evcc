package plugin

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSetter struct{ vals []int64 }

func (r *recordingSetter) IntSetter(string) (func(int64) error, error) {
	return func(v int64) error {
		r.vals = append(r.vals, v)
		return nil
	}, nil
}

var fallbackRecorder = new(recordingSetter)

func init() {
	registry.AddCtx("fallback-test-recorder", func(context.Context, map[string]any) (Plugin, error) {
		return fallbackRecorder, nil
	})
}

func TestFallbackValue(t *testing.T) {
	p, err := NewFallbackFromConfig(t.Context(), map[string]any{
		"value":    3000,
		"fallback": map[string]any{"source": "error", "error": "must not be read"},
	})
	require.NoError(t, err)

	g, err := p.(IntGetter).IntGetter()
	require.NoError(t, err)

	v, err := g()
	require.NoError(t, err)
	assert.EqualValues(t, 3000, v)
}

func TestFallbackSource(t *testing.T) {
	for _, value := range []any{nil, "", 0} {
		p, err := NewFallbackFromConfig(t.Context(), map[string]any{
			"value":    value,
			"fallback": map[string]any{"source": "const", "value": 3680},
		})
		require.NoError(t, err)

		g, err := p.(FloatGetter).FloatGetter()
		require.NoError(t, err)

		v, err := g()
		require.NoError(t, err)
		assert.Equal(t, 3680.0, v)
	}
}

func TestFallbackReadsOnceAndRetriesErrors(t *testing.T) {
	var calls int
	p := &fallbackPlugin{scale: 1, source: func() (float64, error) {
		calls++
		if calls == 1 {
			return 0, errors.New("boom")
		}
		return 3680, nil
	}}

	_, err := p.value()
	require.Error(t, err)

	for range 3 {
		v, err := p.value()
		require.NoError(t, err)
		assert.Equal(t, 3680.0, v)
	}
	assert.Equal(t, 2, calls)
}

func TestFallbackSetter(t *testing.T) {
	fallbackRecorder.vals = nil

	p, err := NewFallbackFromConfig(t.Context(), map[string]any{
		"fallback": map[string]any{"source": "const", "value": 3680},
		"scale":    -1,
		"set":      map[string]any{"source": "fallback-test-recorder"},
	})
	require.NoError(t, err)

	set, err := p.(IntSetter).IntSetter("mode")
	require.NoError(t, err)

	// the passed value is ignored, the resolved value is written
	require.NoError(t, set(1))
	require.NoError(t, set(2))
	assert.Equal(t, []int64{-3680, -3680}, fallbackRecorder.vals)
}

type intOnlyGetter struct{}

func (intOnlyGetter) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) { return 3900, nil }, nil
}

func init() {
	registry.AddCtx("fallback-test-int", func(context.Context, map[string]any) (Plugin, error) {
		return intOnlyGetter{}, nil
	})
}

func TestFallbackIntSource(t *testing.T) {
	p, err := NewFallbackFromConfig(t.Context(), map[string]any{
		"fallback": map[string]any{"source": "fallback-test-int"},
	})
	require.NoError(t, err)

	g, err := p.(FloatGetter).FloatGetter()
	require.NoError(t, err)

	v, err := g()
	require.NoError(t, err)
	assert.Equal(t, 3900.0, v)
}

func TestFallbackGoIntSource(t *testing.T) {
	// go implements both getters but only the int getter accepts an int64 result
	p, err := NewFallbackFromConfig(t.Context(), map[string]any{
		"fallback": map[string]any{"source": "go", "script": "int64(3900)"},
	})
	require.NoError(t, err)

	g, err := p.(IntGetter).IntGetter()
	require.NoError(t, err)

	v, err := g()
	require.NoError(t, err)
	assert.Equal(t, int64(3900), v)
}

func TestFallbackInvalidValues(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "-Inf"} {
		_, err := NewFallbackFromConfig(t.Context(), map[string]any{"value": value})
		assert.Error(t, err, value)
	}

	_, err := NewFallbackFromConfig(t.Context(), map[string]any{"value": 1, "scale": "NaN"})
	assert.Error(t, err)

	// non-finite readings are rejected and not kept
	var calls int
	p := &fallbackPlugin{scale: 1, source: func() (float64, error) {
		calls++
		if calls == 1 {
			return math.NaN(), nil
		}
		return 3680, nil
	}}

	_, err = p.value()
	require.Error(t, err)

	v, err := p.value()
	require.NoError(t, err)
	assert.Equal(t, 3680.0, v)

	// zero readings mean unavailable and are not kept
	calls = 0
	p = &fallbackPlugin{scale: 1, source: func() (float64, error) {
		calls++
		if calls == 1 {
			return 0, nil
		}
		return 3680, nil
	}}

	_, err = p.value()
	require.Error(t, err)

	v, err = p.value()
	require.NoError(t, err)
	assert.Equal(t, 3680.0, v)

	// scaled values must stay finite
	p = &fallbackPlugin{scale: 2, val: 1e308, read: true}
	_, err = p.value()
	assert.Error(t, err)

	// values outside the int64 range are rejected
	p = &fallbackPlugin{scale: 1, val: 1e300, read: true}
	_, err = p.intValue()
	assert.Error(t, err)
}

func TestFallbackMissing(t *testing.T) {
	_, err := NewFallbackFromConfig(t.Context(), map[string]any{})
	assert.Error(t, err)

	p, err := NewFallbackFromConfig(t.Context(), map[string]any{"value": 1})
	require.NoError(t, err)

	_, err = p.(IntSetter).IntSetter("mode")
	assert.Error(t, err)
}
