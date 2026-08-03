package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

// TestPlannerRateGap verifies detection of an undefined-tariff window (a demand
// window left undefined), both interior and leading (forward-pruned rates), vs.
// covered slots, static/no tariff and the region beyond the published horizon.
func TestPlannerRateGap(t *testing.T) {
	base := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	at := func(h int) time.Time { return base.Add(time.Duration(h) * time.Hour) }

	// rates cover 08:00-15:00 and 21:00-24:00, leaving 15:00-21:00 as a gap;
	// includes a zero-price slot to prove a gap differs from a 0 value
	rates := api.Rates{
		{Start: at(8), End: at(12), Value: 0.20},
		{Start: at(12), End: at(15), Value: 0}, // zero-price, but covered
		{Start: at(21), End: at(24), Value: 0.10},
	}

	tc := []struct {
		name  string
		rates api.Rates
		t     time.Time
		want  bool
	}{
		{"interior gap (demand window)", rates, at(18), true},
		{"covered slot", rates, at(9), false},
		{"covered zero-price slot", rates, at(13), false},
		{"gap edge start", rates, at(15), true}, // 15:00 not covered (slot ends at 15:00)
		{"gap edge end", rates, at(20), true},   // still before 21:00 slot
		// planner rates are forward-pruned, so a t before the first slot means the
		// current period is undefined (a leading demand-window gap) - treat as gap
		{"leading gap (undefined now)", rates, at(7), true},
		{"beyond horizon tail", rates, at(24), false}, // at/after last slot end
		{"no rates", nil, at(18), false},
		{"empty rates", api.Rates{}, at(18), false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, plannerRateGap(tc.rates, tc.t))
		})
	}
}
