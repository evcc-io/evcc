package cmd

import (
	"errors"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunOrdered(t *testing.T) {
	var mu sync.Mutex
	var order []string
	record := func(name string) func() error {
		return func() error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, name)
			return nil
		}
	}

	t.Run("references run after their targets", func(t *testing.T) {
		order = nil
		require.NoError(t, runOrdered([]orderedItem{
			{name: "parent", refs: []string{"relay", "unknown"}, run: record("parent")},
			{name: "relay", run: record("relay")},
			{name: "grandparent", refs: []string{"parent"}, run: record("grandparent")},
		}))
		assert.Equal(t, []string{"relay", "parent", "grandparent"}, order)
	})

	t.Run("cycle", func(t *testing.T) {
		order = nil
		err := runOrdered([]orderedItem{
			{name: "a", refs: []string{"b"}, run: record("a")},
			{name: "b", refs: []string{"a"}, run: record("b")},
		})
		require.Error(t, err)
		assert.Empty(t, order)
	})

	t.Run("later waves still run after an error", func(t *testing.T) {
		order = nil
		failed := errors.New("boom")
		err := runOrdered([]orderedItem{
			{name: "relay", run: func() error { return failed }},
			{name: "parent", refs: []string{"relay"}, run: record("parent")},
		})
		assert.ErrorIs(t, err, failed)
		assert.Equal(t, []string{"parent"}, order)
	})
}

func TestStaticRefs(t *testing.T) {
	cc := config.Named{Name: "heatpump", Other: map[string]any{
		"icon":  "heatpump",
		"boost": map[string]any{"type": "device", "name": "relay"},
		"dim":   map[string]any{"type": "shelly", "host": "relay"},
	}}
	assert.Equal(t, []string{"relay"}, staticRefs(cc))
}
