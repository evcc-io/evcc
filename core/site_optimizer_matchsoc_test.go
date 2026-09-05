package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/tariff"
	"github.com/stretchr/testify/assert"
)

func TestMatchSoc(t *testing.T) {
	atLeast50 := func(soc float32) bool { return soc >= 50 }

	// 10 minutes into the 12:00-12:15 slot
	now := time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC)
	eos := time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC)

	for _, tc := range []struct {
		ts       []float32
		expected time.Time
	}{
		// slot 0 is the remainder of the current slot, it ends at the boundary
		{[]float32{50, 0, 0}, eos},
		{[]float32{0, 50, 0}, eos.Add(tariff.SlotDuration)},
		{[]float32{0, 0, 50}, eos.Add(2 * tariff.SlotDuration)},
		// no match
		{[]float32{0, 0, 0}, time.Time{}},
		{nil, time.Time{}},
	} {
		assert.Equal(t, tc.expected, matchSoc(tc.ts, now, atLeast50), "%v", tc.ts)
	}

	// called exactly on a slot boundary the current slot is full length
	onBoundary := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, onBoundary.Add(tariff.SlotDuration), matchSoc([]float32{50}, onBoundary, atLeast50))
}
