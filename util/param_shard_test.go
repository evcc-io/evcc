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
	}, maps.Collect(s.AllShards()), "non-cached")

	assert.Equal(t, map[string]any{
		"A": "a",
		"B": "b",
	}, maps.Collect(s.ModifiedShards()), "cache cold")

	assert.Equal(t, map[string]any{}, maps.Collect(s.ModifiedShards()), "cache warm")

	s = NewSharder("foo", S{"a", "c"})

	assert.Equal(t, map[string]any{
		"B": "c",
	}, maps.Collect(s.ModifiedShards()), "cache modied")
}
