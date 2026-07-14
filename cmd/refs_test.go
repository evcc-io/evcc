package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

func TestCollectCircuitRefs(t *testing.T) {
	references.meter = nil

	circuits := []config.Named{
		{Name: "main", Other: map[string]any{"maxPower": 110000, "meter": "grid"}},
		{Name: "sub", Other: map[string]any{"parent": "main", "maxPower": 43000, "meter": "subfeed"}},
		{Name: "noMeter", Other: map[string]any{"maxPower": 1000}},
	}

	assert.NoError(t, collectCircuitRefs(circuits))
	assert.Equal(t, []string{"grid", "subfeed", ""}, references.meter)
}
