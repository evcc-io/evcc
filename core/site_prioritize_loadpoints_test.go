package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

// TestPrioritizedLoadpoints checks the update-round order: fast/deadline-bound
// first, then descending priority, stable within a tier.
func TestPrioritizedLoadpoints(t *testing.T) {
	mk := func(prio int, fast bool) *Loadpoint {
		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.priority = prio
		if fast {
			lp.mode = api.ModeNow
		}
		return lp
	}

	low := mk(1, false)
	high := mk(5, false)
	forced := mk(0, true) // fast charging outranks priority

	site := &Site{loadpoints: []*Loadpoint{low, high, forced}}
	require.Equal(t, []*Loadpoint{forced, high, low}, site.prioritizedLoadpoints())
}
