package planner

import (
	"math/rand"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitPrecondition(t *testing.T) {
	clock := clock.NewMock()
	rr := rates([]float64{1, 2, 3, 4}, clock.Now(), tariff.SlotDuration)
	rates, precond := splitPreconditionSlots(rr, clock.Now().Add(3*tariff.SlotDuration))
	assert.Equal(t, rr[0:3], rates, "rates")
	assert.Equal(t, rr[3:], precond, "precond")
}

func TestSlotHasSuccessor(t *testing.T) {
	plan := rates([]float64{20, 60, 10, 80, 40, 90}, time.Now(), time.Hour)

	last := plan[len(plan)-1]
	rand.Shuffle(len(plan)-1, func(i, j int) {
		plan[i], plan[j] = plan[j], plan[i]
	})

	for i := range plan {
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

func TestDuration(t *testing.T) {
	now := time.Now()
	plan := api.Rates{
		{Start: now, End: now.Add(time.Hour)},
		{Start: now.Add(time.Hour), End: now.Add(time.Hour)}, // zero - without impact
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)},
	}
	require.Equal(t, 2*time.Hour, Duration(plan))
	require.Equal(t, time.Duration(0), Duration(api.Rates{}))
}

func TestAverageCost(t *testing.T) {
	now := time.Now()
	plan := api.Rates{
		{Start: now, End: now.Add(30 * time.Minute), Value: 10.0},                    // 0.5h * 10 = 5
		{Start: now, End: now, Value: 999.0},                                         // zero - ignored
		{Start: now.Add(30 * time.Minute), End: now.Add(2 * time.Hour), Value: 20.0}, // 1.5h * 20 = 30
	}
	require.Equal(t, 17.5, AverageCost(plan)) // (5 + 30) / 2h = 17.5
	require.Equal(t, 0.0, AverageCost(api.Rates{}))
	require.Equal(t, 0.0, AverageCost(api.Rates{{Start: now, End: now, Value: 10}}))
}

func TestStartEnd(t *testing.T) {
	now := time.Now()
	plan := api.Rates{
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)},
		{Start: now, End: now.Add(time.Hour)},
	}
	require.Equal(t, now, Start(plan))
	require.Equal(t, now.Add(3*time.Hour), End(plan))
	require.True(t, Start(api.Rates{}).IsZero())
	require.True(t, End(api.Rates{}).IsZero())
}

func TestSlotAt(t *testing.T) {
	now := time.Now()
	plan := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 1},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 2},
	}
	require.Equal(t, 1.0, SlotAt(now.Add(30*time.Minute), plan).Value)
	require.Equal(t, 2.0, SlotAt(now.Add(90*time.Minute), plan).Value)
	require.True(t, SlotAt(now.Add(3*time.Hour), plan).IsZero())
}

// TestFindContinuousWindowFlatRates ensures that equal-cost windows always select the
// latest slot. The current slot is clamped to now, hence its cost is summed from
// different addends than the aligned slots and must not decide the comparison.
func TestFindContinuousWindowFlatRates(t *testing.T) {
	day := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rr := rates(lo.RepeatBy(3*96, func(int) float64 { return 0.291 }), day, tariff.SlotDuration)
	target := day.Add(2 * 24 * time.Hour).Add(6*time.Hour + 45*time.Minute)

	for _, tc := range []struct {
		now      time.Duration
		duration time.Duration
	}{
		{16*time.Hour + 45*time.Minute + 35*time.Second, 15*time.Minute + 30*time.Second + 464868247},
		{16*time.Hour + 45*time.Minute + 37*time.Second, 15*time.Minute + 30*time.Second + 239819456},
		{16*time.Hour + 46*time.Minute + 2*time.Second, 26*time.Minute + 39*time.Second + 438201128},
	} {
		now := day.Add(tc.now)

		plan := findContinuousWindow(clampRates(rr, now, target), tc.duration, target)
		require.NotEmpty(t, plan)

		latest := target.Add(-tc.duration).Truncate(tariff.SlotDuration)
		assert.Equal(t, latest, Start(plan), "now %v, duration %v", now, tc.duration)
	}
}

func BenchmarkFindContinuousWindow(b *testing.B) {
	rr := rates(lo.RepeatBy(96, func(i int) float64 {
		return float64(i)
	}), time.Now(), tariff.SlotDuration)

	for b.Loop() {
		findContinuousWindow(rr, 4*tariff.SlotDuration, rr[len(rr)-1].End)
	}
}

func BenchmarkOptimalPlan(b *testing.B) {
	rr := rates(lo.RepeatBy(96, func(i int) float64 {
		return float64(i)
	}), time.Now(), tariff.SlotDuration)

	for b.Loop() {
		optimalPlan(rr, 4*tariff.SlotDuration, rr[len(rr)-1].End)
	}
}
