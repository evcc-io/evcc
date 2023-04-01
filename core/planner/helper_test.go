package planner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSlotHasPreDecessor(t *testing.T) {
	plan := rates([]float64{20, 60, 10, 80, 40, 90}, time.Now(), time.Hour)
	for i := 1; i < len(plan); i++ {
		require.True(t, SlotHasPredecessor(plan[i], plan))
	}
	require.False(t, SlotHasPredecessor(plan[0], plan))
}

func TestSlotHasSuccessor(t *testing.T) {
	plan := rates([]float64{20, 60, 10, 80, 40, 90}, time.Now(), time.Hour)
	for i := 0; i < len(plan)-1; i++ {
		require.True(t, SlotHasSuccessor(plan[i], plan))
	}
	require.False(t, SlotHasSuccessor(plan[len(plan)-1], plan))
}
