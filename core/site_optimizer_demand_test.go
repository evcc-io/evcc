package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClearDemandWhenFull(t *testing.T) {
	for _, tc := range []struct {
		name     string
		demand   []float32
		headroom float32
		expected []float32
	}{
		{
			// 345Wh demanded per slot stores 310.5Wh, so 1000Wh of headroom is covered
			// during the fourth slot- that slot still binds, the ones after it cannot
			"cut once the accumulated energy fills the vehicle",
			[]float32{345, 345, 345, 345, 345, 345},
			1000,
			[]float32{345, 345, 345, 345, 0, 0},
		},
		{
			"headroom beyond the horizon leaves the demand untouched",
			[]float32{345, 345, 345},
			1e6,
			[]float32{345, 345, 345},
		},
		{
			// already at the soc limit: every slot is relaxed at s_max anyway
			"no headroom clears the whole demand",
			[]float32{345, 345, 345},
			0,
			[]float32{0, 0, 0},
		},
		{
			// the shortened first slot carries less energy and must not pull the cut in
			"partial first slot counts with its own energy",
			[]float32{100, 345, 345, 345},
			600,
			[]float32{100, 345, 345, 0},
		},
		{
			"empty demand stays empty",
			[]float32{},
			1000,
			[]float32{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, clearDemandWhenFull(tc.demand, tc.headroom))
		})
	}
}

// the request contract requires every series to have the same length
func TestClearDemandWhenFullKeepsLength(t *testing.T) {
	demand := make([]float32, 96)
	for i := range demand {
		demand[i] = 345
	}

	assert.Len(t, clearDemandWhenFull(demand, 1000), len(demand))
	assert.Len(t, clearDemandWhenFull(demand, 0), len(demand))
}
