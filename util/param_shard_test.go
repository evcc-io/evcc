package util

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSharder(t *testing.T) {
	type S struct {
		A, B string
	}

	s := NewSharder("foo", S{"a", "b"})

	assert.Equal(t, map[string]any{
		"A": "a",
		"B": "b",
	}, maps.Collect(s.Shards(false)), "non-cached")

	assert.Equal(t, map[string]any{
		"A": "a",
		"B": "b",
	}, maps.Collect(s.Shards(true)), "cache cold")

	assert.Equal(t, map[string]any{}, maps.Collect(s.Shards(true)), "cache warm")
}
