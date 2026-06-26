package plugin

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func setCache(t *testing.T, key string, val any) {
	t.Helper()
	c := util.NewParamCache()
	c.Add(key, util.Param{Key: key, Val: val})
	util.SetDefaultParamCache(c)
	t.Cleanup(func() { util.SetDefaultParamCache(nil) })
}

func TestStateGetter(t *testing.T) {
	// nested structure like evopt-batteries[i].suggestion.charge
	setCache(t, "evopt-batteries", []map[string]any{
		{"suggestion": map[string]any{"charge": 0.0}},
		{"suggestion": map[string]any{"charge": 1500.0}},
	})

	for _, tc := range []struct {
		jq    string
		scale float64
		want  float64
	}{
		{"[.[] | .suggestion.charge][0]", 1, 0},
		{"[.[] | .suggestion.charge][1]", 1, 1500},
		{"[.[] | .suggestion.charge][1]", 0.001, 1.5},
	} {
		other := map[string]any{"key": "evopt-batteries", "jq": tc.jq}
		if tc.scale != 1 {
			other["scale"] = tc.scale
		}
		p, err := NewStateFromConfig(t.Context(), other)
		require.NoError(t, err)

		g, err := p.(FloatGetter).FloatGetter()
		require.NoError(t, err)
		v, err := g()
		require.NoError(t, err)
		require.Equal(t, tc.want, v, "jq %q scale %v", tc.jq, tc.scale)
	}
}

func TestStateGetterFlatAndMissing(t *testing.T) {
	setCache(t, "arr", []int64{0, 1500, 3000})

	p, err := NewStateFromConfig(t.Context(), map[string]any{"key": "arr", "jq": ".[2]"})
	require.NoError(t, err)
	g, _ := p.(FloatGetter).FloatGetter()
	v, err := g()
	require.NoError(t, err)
	require.Equal(t, 3000.0, v)

	// unknown key -> 0
	p, err = NewStateFromConfig(t.Context(), map[string]any{"key": "missing", "jq": ".[0]"})
	require.NoError(t, err)
	g, _ = p.(FloatGetter).FloatGetter()
	v, err = g()
	require.NoError(t, err)
	require.Equal(t, 0.0, v)

	// jq resolving to null without a default -> error (surfaces a mistyped key/filter
	// instead of silently returning a value)
	setCache(t, "obj", []map[string]any{{"foo": 1.0}})
	p, err = NewStateFromConfig(t.Context(), map[string]any{"key": "obj", "jq": ".[0].missing.charge"})
	require.NoError(t, err)
	g, _ = p.(FloatGetter).FloatGetter()
	_, err = g()
	require.Error(t, err, "null result without an explicit default must error")

	// defaulting is consumer policy: express it in the jq -> 0
	p, err = NewStateFromConfig(t.Context(), map[string]any{"key": "obj", "jq": ".[0].missing.charge // 0"})
	require.NoError(t, err)
	g, _ = p.(FloatGetter).FloatGetter()
	v, err = g()
	require.NoError(t, err)
	require.Equal(t, 0.0, v)
}

// TestStateIntSetterForward verifies the nested-setter path: the incoming int is
// ignored, the jq-selected cached value is read and forwarded to the nested setter.
func TestStateIntSetterForward(t *testing.T) {
	setCache(t, "evopt-batteries", []map[string]any{
		{"suggestion": map[string]any{"charge": 1500.0}},
	})

	var mu sync.Mutex
	var written string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		written = r.URL.Query().Get("v")
		mu.Unlock()
	}))
	defer srv.Close()

	p, err := NewStateFromConfig(t.Context(), map[string]any{
		"key": "evopt-batteries",
		"jq":  ".[0].suggestion.charge",
		"set": map[string]any{"source": "http", "uri": srv.URL + "/write?v={{.foo}}"},
	})
	require.NoError(t, err)

	set, err := p.(IntSetter).IntSetter("foo")
	require.NoError(t, err)

	// incoming value (99) is ignored; cached suggestion.charge (1500) is forwarded
	require.NoError(t, set(99))

	mu.Lock()
	got, _ := strconv.ParseFloat(written, 64)
	mu.Unlock()
	require.Equal(t, 1500.0, got)
}
