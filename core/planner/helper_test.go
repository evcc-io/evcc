package planner

import (
	"math/rand"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestSlotHasSuccessor(t *testing.T) {
	plan := rates([]float64{20, 60, 10, 80, 40, 90}, time.Now(), time.Hour)

	last := plan[len(plan)-1]
	rand.Shuffle(len(plan)-1, func(i, j int) {
		plan[i], plan[j] = plan[j], plan[i]
	})

	for i := 0; i < len(plan); i++ {
		if plan[i] != last {
			require.True(t, SlotHasSuccessor(plan[i], plan))
		}
	}

	require.False(t, SlotHasSuccessor(last, plan))
}

func TestIsFirst(t *testing.T) {
	clock := clock.NewMock()
	plan := rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour)

	first := plan[0]
	rand.Shuffle(len(plan), func(i, j int) {
		plan[i], plan[j] = plan[j], plan[i]
	})

	for i := 1; i < len(plan); i++ {
		if plan[i] != first {
			require.False(t, IsFirst(plan[i], plan))
		}
	}

	require.True(t, IsFirst(first, plan))

	// ensure single slot is always first
	require.True(t, IsFirst(first, []api.Rate{first}))
}
