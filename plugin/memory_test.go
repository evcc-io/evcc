package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func memoryFloatSetter(t *testing.T, ctx context.Context, other map[string]any) func(float64) error {
	t.Helper()
	p, err := NewMemoryFromConfig(ctx, other)
	require.NoError(t, err)
	set, err := p.(FloatSetter).FloatSetter("x")
	require.NoError(t, err)
	return set
}

func memoryFloatGetter(t *testing.T, ctx context.Context, name string) func() (float64, error) {
	t.Helper()
	p, err := NewMemoryFromConfig(ctx, map[string]any{"name": name})
	require.NoError(t, err)
	get, err := p.(FloatGetter).FloatGetter()
	require.NoError(t, err)
	return get
}

// TestMemorySinkForward verifies the two roles over a shared device store: a sink stores
// the incoming value, and a forward (with nested set) ignores its input, reads the stored
// value and writes it to the nested setter - the site-push -> control-path handoff.
func TestMemorySinkForward(t *testing.T) {
	ctx := WithMemoryStore(t.Context())

	// sink: site push target, stores into cell "cap"
	sink := memoryFloatSetter(t, ctx, map[string]any{"name": "cap"})

	// forward: reads cell "cap", writes it to cell "out" via a nested memory sink
	fwdP, err := NewMemoryFromConfig(ctx, map[string]any{
		"name": "cap",
		"set":  map[string]any{"source": "memory", "name": "out"},
	})
	require.NoError(t, err)
	// consumed by an int control path (batterymode switch): incoming mode value is ignored
	fwd, err := fwdP.(IntSetter).IntSetter("x")
	require.NoError(t, err)

	out := memoryFloatGetter(t, ctx, "out")

	// before any push the forward writes the zero value
	require.NoError(t, fwd(4))
	v, err := out()
	require.NoError(t, err)
	require.Equal(t, 0.0, v)

	// site pushes 1500 into "cap"; forward now propagates it to "out"
	require.NoError(t, sink(1500))
	require.NoError(t, fwd(4))
	v, err = out()
	require.NoError(t, err)
	require.Equal(t, 1500.0, v)
}

// TestMemoryDeviceIsolation verifies that stores scoped to different contexts (i.e.
// different devices) do not share cells of the same name.
func TestMemoryDeviceIsolation(t *testing.T) {
	ctxA := WithMemoryStore(t.Context())
	ctxB := WithMemoryStore(t.Context())

	memoryFloatSetter(t, ctxA, map[string]any{"name": "cap"})(1500)

	require.Equal(t, 1500.0, mustGet(t, memoryFloatGetter(t, ctxA, "cap")), "device A sees its own value")
	require.Equal(t, 0.0, mustGet(t, memoryFloatGetter(t, ctxB, "cap")), "device B is isolated")
}

func TestMemoryMissingName(t *testing.T) {
	_, err := NewMemoryFromConfig(t.Context(), map[string]any{})
	require.Error(t, err)
}

// TestMemoryInitial verifies that initial seeds the cell before any push, so a forward
// reading it early (e.g. a control path applied before the first site push cycle) sees
// the configured fallback instead of the zero value - and that a later push overwrites it.
func TestMemoryInitial(t *testing.T) {
	ctx := WithMemoryStore(t.Context())

	initial := 3000.0
	_, err := NewMemoryFromConfig(ctx, map[string]any{"name": "cap", "initial": initial})
	require.NoError(t, err)

	require.Equal(t, 3000.0, mustGet(t, memoryFloatGetter(t, ctx, "cap")), "seeded before any push")

	memoryFloatSetter(t, ctx, map[string]any{"name": "cap"})(1500)
	require.Equal(t, 1500.0, mustGet(t, memoryFloatGetter(t, ctx, "cap")), "push overwrites the seeded value")
}

func mustGet(t *testing.T, get func() (float64, error)) float64 {
	t.Helper()
	v, err := get()
	require.NoError(t, err)
	return v
}
