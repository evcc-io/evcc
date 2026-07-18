package planner

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sum of reserved power per slot across all plans must never exceed the budget
func assertFeasible(t *testing.T, budget float64, maxPower float64, plans []api.Rates) {
	used := map[time.Time]float64{}
	for _, p := range plans {
		for _, s := range p {
			for ts := s.Start.Truncate(time.Hour); ts.Before(s.End); ts = ts.Add(time.Hour) {
				used[ts] += maxPower
			}
		}
	}
	for ts, u := range used {
		assert.LessOrEqualf(t, u, budget, "slot %v over budget", ts)
	}
}

func TestAllocateShared(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	target := start.Add(6 * time.Hour)

	rates := func() api.Rates {
		var r api.Rates
		for i := 0; i < 6; i++ {
			s := start.Add(time.Duration(i) * time.Hour)
			r = append(r, api.Rate{Start: s, End: s.Add(time.Hour), Value: 0.1})
		}
		return r
	}

	const minP = 1380.0

	t.Run("full-power loadpoints cannot share a slot", func(t *testing.T) {
		// two 11kW loadpoints on an 11kW circuit -> must take different slots
		reqs := []SharedPlanRequest{
			{Priority: 0, MaxPower: 11000, MinPower: minP, RequiredDuration: time.Hour, TargetTime: target},
			{Priority: 0, MaxPower: 11000, MinPower: minP, RequiredDuration: time.Hour, TargetTime: target},
		}
		plans := AllocateShared(start, 11000, rates(), reqs)
		require.Len(t, plans, 2)
		assert.Equal(t, time.Hour, Duration(plans[0]))
		assert.Equal(t, time.Hour, Duration(plans[1]))
		assert.False(t, plans[0][0].Start.Equal(plans[1][0].Start), "must not share the slot")
		assertFeasible(t, 11000, 11000, plans)
	})

	t.Run("small loadpoints share a slot", func(t *testing.T) {
		// two 3.7kW loadpoints on 11kW -> both fit the same cheapest slot
		reqs := []SharedPlanRequest{
			{Priority: 0, MaxPower: 3700, MinPower: minP, RequiredDuration: time.Hour, TargetTime: target},
			{Priority: 0, MaxPower: 3700, MinPower: minP, RequiredDuration: time.Hour, TargetTime: target},
		}
		plans := AllocateShared(start, 11000, rates(), reqs)
		assert.True(t, plans[0][0].Start.Equal(plans[1][0].Start), "should share the cheapest slot")
		assertFeasible(t, 11000, 3700, plans)
	})

	t.Run("forced load is placed before higher priority", func(t *testing.T) {
		// forced low-priority reserves the cheapest slot ahead of a high-priority peer
		// one clearly-cheapest slot (index 2) both want
		cheap := rates()
		cheap[2].Value = 0.05
		cheapStart := cheap[2].Start

		reqs := []SharedPlanRequest{
			{Priority: 9, MaxPower: 11000, MinPower: minP, RequiredDuration: time.Hour, TargetTime: target}, // high prio
			{Forced: true, Priority: 0, MaxPower: 11000, MinPower: minP, RequiredDuration: time.Hour, TargetTime: target},
		}
		plans := AllocateShared(start, 11000, cheap, reqs)
		// forced (index 1) grabs the cheapest slot ahead of the high-priority peer
		assert.True(t, plans[1][0].Start.Equal(cheapStart), "forced load takes the cheapest slot")
		assert.False(t, plans[0][0].Start.Equal(cheapStart), "high-priority peer yields the cheapest slot")
		assertFeasible(t, 11000, 11000, plans)
	})

	t.Run("does not schedule past the target", func(t *testing.T) {
		// target in 2h but rates span 6h: no slot may start at/after the target
		near := start.Add(2 * time.Hour)
		reqs := []SharedPlanRequest{
			{MaxPower: 11000, MinPower: minP, RequiredDuration: time.Hour, TargetTime: near},
		}
		plans := AllocateShared(start, 11000, rates(), reqs)
		require.Equal(t, time.Hour, Duration(plans[0]))
		for _, s := range plans[0] {
			assert.False(t, s.Start.Before(start), "slot before now")
			assert.False(t, !s.End.After(start) || s.Start.After(near) || s.End.After(near), "slot outside [now,target]")
		}
	})
}
