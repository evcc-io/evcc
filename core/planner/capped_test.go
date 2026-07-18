package planner

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestPlanCapped(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	target := start.Add(4 * time.Hour)

	mk := func(n int) api.Rates {
		var r api.Rates
		for i := 0; i < n; i++ {
			s := start.Add(time.Duration(i) * time.Hour)
			r = append(r, api.Rate{Start: s, End: s.Add(time.Hour), Value: 0.1})
		}
		return r
	}

	const maxPower, minPower = 11000.0, 1380.0
	full := func(time.Time) float64 { return maxPower }
	half := func(time.Time) float64 { return maxPower / 2 }

	// full availability == unconstrained duration accounting
	p := planCapped(mk(4), time.Hour, target, full, maxPower, minPower)
	assert.Equal(t, time.Hour, Duration(p))

	// half power -> plan spills into twice the wall-clock time for the same energy
	p = planCapped(mk(4), time.Hour, target, half, maxPower, minPower)
	assert.Equal(t, 2*time.Hour, Duration(p))
	assert.Len(t, p, 2)

	// a slot below min power is skipped (semi-continuous)
	skipFirst := func(ts time.Time) float64 {
		if ts.Equal(start) {
			return minPower - 1
		}
		return maxPower
	}
	p = planCapped(mk(4), time.Hour, target, skipFirst, maxPower, minPower)
	assert.Equal(t, time.Hour, Duration(p))
	for _, r := range p {
		assert.False(t, r.Start.Equal(start), "sub-min slot must be skipped")
	}
}
