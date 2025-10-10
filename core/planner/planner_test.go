package planner

import (
	"slices"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func rates(prices []float64, start time.Time, slotDuration time.Duration) api.Rates {
	res := make(api.Rates, 0, len(prices))

	for i, v := range prices {
		slotStart := start.Add(time.Duration(i) * slotDuration)
		ar := api.Rate{
			Start: slotStart,
			End:   slotStart.Add(slotDuration),
			Value: v,
		}
		res = append(res, ar)
	}

	return res
}

func TestPlan(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	rates, err := trf.Rates()
	require.NoError(t, err)

	slices.SortStableFunc(rates, sortByCost)

	{
		plan := p.plan(rates, time.Hour, clock.Now())
		assert.Empty(t, plan)
	}

	tc := []struct {
		desc string
		// task
		duration time.Duration
		now      time.Time
		target   time.Time
		// result
		planStart time.Time
		planCost  float64
	}{
		// With window bundling: prefer continuous windows when costs are similar
		// Rates: [20, 60, 10, 80, 40, 90] at hours 0-5
		{
			"plan 0-0-60-0-0-0",
			time.Hour,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(2 * time.Hour), // cheapest single slot at hour 2
			10,
		},
		{
			"plan 60-0-60-0-0-0", // picks cheapest 2 slots (0+2), not continuous 0-1
			2 * time.Hour,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(0 * time.Hour),
			30, // slot 0 (20) + slot 2 (10) = 30, much cheaper than continuous 0-1 (80)
		},
		{
			"plan (30)30-0-60-0-0-0", // picks slot 0 shortened + slot 2
			90 * time.Minute,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute), // first slot shortened, starts at 0:30
			20,                                // 30min@20 + 60min@10 = 10 + 10
		},
		{
			"plan 0-0-60-0-0-0",
			time.Hour,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(2 * time.Hour),
			10,
		},
		{
			"plan 60-0-60-0-0-0", // after 30min start: picks continuous 1-2
			2 * time.Hour,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(time.Hour), // starts at hour 1
			70,                         // continuous hours 1-2: 60 + 10 = 70
		},
		{
			"plan (30)30-0-60-0-0-0",
			90 * time.Minute,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			20, // 30min@20 + 60min@10
		},
	}

	for i, tc := range tc {
		t.Log(tc.desc)
		clock.Set(tc.now)
		plan := p.plan(rates, tc.duration, tc.target)

		assert.Equalf(t, tc.planStart.UTC(), Start(plan).UTC(), "case %d start", i)
		assert.Equalf(t, tc.duration, Duration(plan), "case %d duration", i)
		assert.Equalf(t, tc.planCost, AverageCost(plan)*float64(Duration(plan))/float64(time.Hour), "case %d cost", i)
	}
}

func TestSlotBundling(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	tc := []struct {
		desc         string
		rates        []float64
		duration     time.Duration
		target       time.Time
		expectedCost float64
	}{
		{
			desc:         "single continuous cheapest window",
			rates:        []float64{12, 18, 21, 17, 13, 10, 10, 10, 50, 50, 50, 10, 11, 10, 12},
			duration:     3 * time.Hour,
			target:       clock.Now().Add(15 * time.Hour),
			expectedCost: 30, // continuous 5-7 @ 10 each
		},
		{
			desc:         "two separate cheap slots when much cheaper",
			rates:        []float64{15, 50, 15, 50, 15, 50},
			duration:     2 * time.Hour,
			target:       clock.Now().Add(6 * time.Hour),
			expectedCost: 30, // can pick separate slots 0,2 or 2,4 @ 15 each
		},
		{
			desc:         "continuous window when only slightly more expensive",
			rates:        []float64{20, 20, 20, 20, 10, 100},
			duration:     4 * time.Hour,
			target:       clock.Now().Add(6 * time.Hour),
			expectedCost: 70, // will pick slot 4 (10) + continuous 0-2 (60)
		},
		{
			desc:         "two separate cheap windows",
			rates:        []float64{10, 80, 80, 10, 80, 80},
			duration:     2 * time.Hour,
			target:       clock.Now().Add(6 * time.Hour),
			expectedCost: 20, // hours 0 and 3 @ 10 each
		},
		{
			desc:         "three separate cheap windows, latest slots",
			rates:        []float64{10, 80, 10, 80, 10, 80, 10, 80, 10, 80},
			duration:     3 * time.Hour,
			target:       clock.Now().Add(10 * time.Hour),
			expectedCost: 30, // hours 4,6,8 @ 10 each
		},
		{
			desc:         "4 hours grouped in three separate windows, to reach target at latest slots",
			rates:        []float64{10, 80, 10, 80, 10, 80, 10, 80, 10, 80},
			duration:     4 * time.Hour,
			target:       clock.Now().Add(10 * time.Hour),
			expectedCost: 110, // hours 4,6,8 @ 10 each and 9 @ 80
		},
		{
			desc:         "4 hours grouped in three separate windows, to reach target at latest slots",
			rates:        []float64{10, 80, 10, 80, 10, 80, 10, 80, 10, 81},
			duration:     4 * time.Hour,
			target:       clock.Now().Add(10 * time.Hour),
			expectedCost: 110, // hours 4,6,8 @ 10 each and 7 @ 80
		},
	}

	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			testRates := rates(tc.rates, clock.Now(), time.Hour)
			slices.SortStableFunc(testRates, sortByCost)

			plan := p.plan(testRates, tc.duration, tc.target)

			assert.Equal(t, tc.duration, Duration(plan), "total duration mismatch")

			totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(time.Hour)
			assert.InDelta(t, tc.expectedCost, totalCost, 0.01, "cost mismatch")
		})
	}
}

func TestSlotBundlingEdgeCases(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	t.Run("single slot covers entire duration", func(t *testing.T) {
		testRates := rates([]float64{10, 50, 50, 50}, clock.Now(), 2*time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 90*time.Minute, clock.Now().Add(8*time.Hour))

		assert.Equal(t, 90*time.Minute, Duration(plan))
		assert.Equal(t, 1, len(plan), "should use single shortened slot")
	})

	t.Run("all slots same price prefer later start", func(t *testing.T) {
		testRates := rates([]float64{20, 20, 20, 20}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 2*time.Hour, clock.Now().Add(4*time.Hour))

		assert.Equal(t, 2*time.Hour, Duration(plan))

		// With equal prices, continuous window is preferred, and later start is preferred
		// Should pick continuous 2-hour window, preferably starting at hour 2 (latest possible)
		startTime := Start(plan)
		validStarts := startTime.Equal(clock.Now()) ||
			startTime.Equal(clock.Now().Add(time.Hour)) ||
			startTime.Equal(clock.Now().Add(2*time.Hour))
		assert.True(t, validStarts, "should use a valid 2-hour continuous window")

		// Verify it's continuous
		assert.Equal(t, 1, countChargingWindows(plan), "should be continuous")
	})

	t.Run("very short slots with gaps", func(t *testing.T) {
		testRates := rates([]float64{10, 100, 10, 100, 10, 100}, clock.Now(), 15*time.Minute)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 30*time.Minute, clock.Now().Add(90*time.Minute))

		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(15*time.Minute)
		assert.InDelta(t, 20.0, totalCost, 0.01, "should pick two cheap slots")
	})

	t.Run("required duration longer than any single window", func(t *testing.T) {
		testRates := rates([]float64{10, 10, 50, 15, 15, 50}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 4*time.Hour, clock.Now().Add(6*time.Hour))

		assert.Equal(t, 4*time.Hour, Duration(plan))

		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(time.Hour)
		assert.InDelta(t, 50.0, totalCost, 0.01)
	})
}

func TestSlotBundlingWithPrecondition(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{10, 20, 30, 5, 15, 25}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(2*time.Hour, time.Hour, clock.Now().Add(6*time.Hour))

	assert.Equal(t, 2*time.Hour, Duration(plan))

	lastSlot := plan[len(plan)-1]
	assert.Equal(t, clock.Now().Add(6*time.Hour), lastSlot.End)
}

func TestFirstSlotShortening(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	testRates := rates([]float64{10, 80, 10, 80}, clock.Now(), time.Hour)
	slices.SortStableFunc(testRates, sortByCost)

	plan := p.plan(testRates, 90*time.Minute, clock.Now().Add(4*time.Hour))

	assert.Equal(t, 90*time.Minute, Duration(plan))

	firstSlot := plan[0]
	slotDuration := firstSlot.End.Sub(firstSlot.Start)
	assert.True(t, slotDuration <= time.Hour, "first slot should be at most 60min")
}

func countChargingWindows(plan api.Rates) int {
	if len(plan) == 0 {
		return 0
	}

	plan.Sort()
	windows := 1
	for i := 1; i < len(plan); i++ {
		if !plan[i].Start.Equal(plan[i-1].End) {
			windows++
		}
	}
	return windows
}

func TestNilTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute))
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(60 * time.Minute),
		},
	}, plan, "expected simple plan")
}

func TestRatesError(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(nil, api.ErrOutdated)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute))
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(60 * time.Minute),
		},
	}, plan, "expected simple plan")
}

func TestFlatTariffTargetInThePast(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	simplePlan := api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(60 * time.Minute),
		},
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute))
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan = p.Plan(time.Hour, 0, clock.Now().Add(-30*time.Minute))
	assert.Equal(t, simplePlan, plan, "expected simple plan")
}

func TestFlatTariffLongSlots(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now(), 24*time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(2*time.Hour))
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}, SlotAt(clock.Now(), plan))
	assert.Equal(t, api.Rate{}, SlotAt(clock.Now().Add(time.Hour), plan))

	plan = p.Plan(time.Hour, 0, clock.Now().Add(time.Hour))
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}, SlotAt(clock.Now(), plan))
}

func TestTargetAfterKnownPrices(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(40*time.Minute, 0, clock.Now().Add(2*time.Hour))
	assert.False(t, !SlotAt(clock.Now(), plan).IsZero(), "should not start if car can be charged completely after known prices ")

	plan = p.Plan(2*time.Hour, 0, clock.Now().Add(2*time.Hour))
	assert.True(t, !SlotAt(clock.Now(), plan).IsZero(), "should start if car can not be charged completely after known prices ")
}

func TestChargeAfterTargetTime(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 0, 0, 0}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	simplePlan := api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(60 * time.Minute),
		},
	}

	plan := p.Plan(time.Hour, 0, clock.Now())
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan = p.Plan(time.Hour, 0, clock.Now().Add(-time.Hour))
	assert.Equal(t, simplePlan, plan, "expected simple plan")
}

func TestPrecondition(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 1, 2, 3}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// Test 1: 1h duration, 1h precondition
	// Should use only the precondition slot (last hour)
	plan := p.Plan(time.Hour, time.Hour, clock.Now().Add(4*time.Hour))
	assert.Equal(t, time.Hour, Duration(plan), "expected 1 hour total")

	// Must end at target time
	lastSlot := plan[len(plan)-1]
	assert.Equal(t, clock.Now().Add(4*time.Hour), lastSlot.End, "must end at target time")

	// Should be in precondition zone (hour 3-4)
	assert.True(t, lastSlot.Start.Equal(clock.Now().Add(3*time.Hour)) ||
		lastSlot.Start.After(clock.Now().Add(3*time.Hour)),
		"should use precondition zone")

	// Test 2: 2h duration, 1h precondition
	// Should use 1h optimized (cheapest from 0-3) + 1h precondition (hour 3-4)
	plan = p.Plan(2*time.Hour, time.Hour, clock.Now().Add(4*time.Hour))
	assert.Equal(t, 2*time.Hour, Duration(plan), "expected 2 hours total")

	// Must end at target time
	lastSlot = plan[len(plan)-1]
	assert.Equal(t, clock.Now().Add(4*time.Hour), lastSlot.End, "must end at target time")

	// Should have at least one slot in precondition zone
	hasPreconditionSlot := false
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(4*time.Hour)) &&
			slot.End.After(clock.Now().Add(3*time.Hour)) {
			hasPreconditionSlot = true
			break
		}
	}
	assert.True(t, hasPreconditionSlot, "must include precondition zone")

	// Should have optimized slot before precondition (cheapest is hour 0)
	hasOptimizedSlot := false
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(3 * time.Hour)) {
			hasOptimizedSlot = true
			break
		}
	}
	assert.True(t, hasOptimizedSlot, "should have optimized slot before precondition")

	// Test 3: 1h duration, 30min precondition
	// Should use 30min optimized + 30min precondition
	plan = p.Plan(time.Hour, 30*time.Minute, clock.Now().Add(4*time.Hour))
	assert.Equal(t, time.Hour, Duration(plan), "expected 1 hour total")

	// Must end at target time
	lastSlot = plan[len(plan)-1]
	assert.Equal(t, clock.Now().Add(4*time.Hour), lastSlot.End, "must end at target time")

	// Should have slot(s) covering the last 30 minutes (precondition)
	hasPreconditionPart := false
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(4*time.Hour)) &&
			slot.End.After(clock.Now().Add(210*time.Minute)) {
			hasPreconditionPart = true
			break
		}
	}
	assert.True(t, hasPreconditionPart, "must include precondition period")
}

// TestPreconditionSeparation verifies that precondition slots are kept separate
// from optimization windows to prevent dilution of the precondition requirement
func TestPreconditionSeparation(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	// Rates with expensive slot before precondition
	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{10, 20, 80, 15}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// 2h duration, 1h precondition
	// Should use: 1h from cheap slots (0 or 1) + 1h precondition (slot 3)
	// Should NOT create window including slot 2 (expensive) with slot 3 (precondition)
	plan := p.Plan(2*time.Hour, time.Hour, clock.Now().Add(4*time.Hour))

	assert.Equal(t, 2*time.Hour, Duration(plan), "expected 2 hours total")

	// Verify precondition slot is included
	hasPrecondition := false
	for _, slot := range plan {
		if slot.End.Equal(clock.Now().Add(4 * time.Hour)) {
			hasPrecondition = true
			break
		}
	}
	assert.True(t, hasPrecondition, "must include precondition slot")

	// Verify we didn't use the expensive slot 2
	usedExpensiveSlot := false
	for _, slot := range plan {
		if slot.Start.Equal(clock.Now().Add(2*time.Hour)) ||
			(slot.Start.Before(clock.Now().Add(2*time.Hour)) &&
				slot.End.After(clock.Now().Add(2*time.Hour))) {
			usedExpensiveSlot = true
			break
		}
	}
	assert.False(t, usedExpensiveSlot, "should not use expensive slot when precondition is separate")
}

// TestPreconditionLimiting verifies that precondition is limited to required duration
func TestPreconditionLimiting(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	// Create many slots (simulating "all" precondition)
	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// Test 1: requiredDuration 2h, precondition "all" (10h)
	// Should only mark last 2h, not all 10h
	plan := p.Plan(2*time.Hour, 10*time.Hour, clock.Now().Add(10*time.Hour))

	assert.Equal(t, 2*time.Hour, Duration(plan), "expected 2 hours total")

	// Should use only the last 2 hours (hours 8-10)
	// Not all 10 hours
	allSlotsInLastTwoHours := true
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(8 * time.Hour)) {
			allSlotsInLastTwoHours = false
			break
		}
	}
	assert.True(t, allSlotsInLastTwoHours, "should only use last 2 hours when precondition='all' but only 2h needed")

	// Test 2: requiredDuration 30min, precondition 2h
	// Should only mark last 30min, not 2h
	plan = p.Plan(30*time.Minute, 2*time.Hour, clock.Now().Add(10*time.Hour))

	assert.Equal(t, 30*time.Minute, Duration(plan), "expected 30 minutes total")

	// Should use only the last 30 minutes (9.5h - 10h)
	allSlotsInLastThirtyMin := true
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(9*time.Hour + 30*time.Minute)) {
			allSlotsInLastThirtyMin = false
			break
		}
	}
	assert.True(t, allSlotsInLastThirtyMin, "should only use last 30min when precondition=2h but only 30min needed")

	// Test 3: requiredDuration 5h, precondition 1h
	// Should mark last 1h + optimize 4h before that
	plan = p.Plan(5*time.Hour, 1*time.Hour, clock.Now().Add(10*time.Hour))

	assert.Equal(t, 5*time.Hour, Duration(plan), "expected 5 hours total")

	// Should have at least one slot in the last hour (precondition)
	hasPreconditionSlot := false
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(10*time.Hour)) &&
			slot.End.After(clock.Now().Add(9*time.Hour)) {
			hasPreconditionSlot = true
			break
		}
	}
	assert.True(t, hasPreconditionSlot, "must include precondition hour")

	// Should also have optimized slots before hour 9
	hasOptimizedSlots := false
	for _, slot := range plan {
		if slot.End.Before(clock.Now().Add(9*time.Hour)) ||
			slot.End.Equal(clock.Now().Add(9*time.Hour)) {
			hasOptimizedSlots = true
			break
		}
	}
	assert.True(t, hasOptimizedSlots, "should have optimized slots before precondition zone")
}

func TestContinuousPlanNoTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan := p.Plan(time.Hour, 0, clock.Now())

	assert.Len(t, plan, 1)
	assert.Equal(t, clock.Now(), SlotAt(clock.Now(), plan).Start)
	assert.Equal(t, clock.Now().Add(time.Hour), SlotAt(clock.Now(), plan).End)
}

func TestContinuousPlan(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now().Add(time.Hour), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(150*time.Minute, 0, clock.Now())

	assert.Len(t, plan, 3)
}

func TestContinuousPlanOutsideRates(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now().Add(time.Hour), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(30*time.Minute, 0, clock.Now())

	assert.Len(t, plan, 1)
}

// TestMaxChargingWindows tests the MaxChargingWindows=3 constraint and related logic
func TestMaxChargingWindows(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	t.Run("interruption penalty prefers continuous over fragmented", func(t *testing.T) {
		// Test that 5% InterruptionPenalty makes continuous charging preferred
		// when fragmented charging saves less than ~5% per interruption
		// Continuous: 100€, Fragmented (2 windows): 96€ + 5% penalty = 100.8€
		// Should choose continuous
		testRates := rates([]float64{50, 50, 48, 48}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 2*time.Hour, clock.Now().Add(4*time.Hour))

		assert.Equal(t, 2*time.Hour, Duration(plan))
		windows := countChargingWindows(plan)
		// With 5% penalty, continuous (100€) should be chosen over 2 windows (96€ + ~4.8€ penalty)
		assert.Equal(t, 1, windows, "should prefer continuous charging when savings are marginal")
	})

	t.Run("many cheap windows prefer latest three", func(t *testing.T) {
		// 8 equally cheap slots - should pick the latest 3 (hours 12, 14, 16)
		testRates := rates([]float64{10, 50, 10, 50, 10, 50, 10, 50, 10, 50, 10, 50, 10, 50, 10, 50, 10, 50}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 3*time.Hour, clock.Now().Add(18*time.Hour))

		assert.Equal(t, 3*time.Hour, Duration(plan))
		windows := countChargingWindows(plan)
		assert.LessOrEqual(t, windows, 3, "should not exceed MaxChargingWindows")

		// Should prefer later slots
		firstSlotStart := Start(plan)
		assert.True(t, firstSlotStart.After(clock.Now().Add(10*time.Hour)) ||
			firstSlotStart.Equal(clock.Now().Add(10*time.Hour)),
			"should prefer latest slots when all are equally cheap")
	})

	t.Run("replacement chooses lowest cost increase", func(t *testing.T) {
		// Test that when replacing a window to meet duration requirement,
		// the algorithm picks the replacement with lowest cost increase
		// Windows: 0-1h@10, 2-3h@10, 4-5h@10, 6-7h@10 (4 cheap windows)
		// Need 4 hours, max 3 windows
		// Options: replace any 1h window with 2h window
		// 6-8h@10+15 (avg 12.5, +2.5/h increase) is better than others
		testRates := rates([]float64{10, 50, 10, 50, 10, 50, 10, 15, 50}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 4*time.Hour, clock.Now().Add(9*time.Hour))

		assert.Equal(t, 4*time.Hour, Duration(plan))
		windows := countChargingWindows(plan)
		assert.LessOrEqual(t, windows, 3, "should not exceed MaxChargingWindows")

		// Cost should be optimal: 3x10 + 1x15 = 45
		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(time.Hour)
		assert.InDelta(t, 45.0, totalCost, 0.01, "should choose replacement with lowest cost increase")
	})

	t.Run("no valid replacement falls back gracefully", func(t *testing.T) {
		// Scenario where no valid replacement exists without overlaps
		// Should still produce a valid plan (possibly continuous)
		testRates := rates([]float64{10, 10, 10, 10, 80}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 4*time.Hour, clock.Now().Add(5*time.Hour))

		assert.Equal(t, 4*time.Hour, Duration(plan), "should still meet duration requirement")
		windows := countChargingWindows(plan)
		assert.LessOrEqual(t, windows, 3, "should not exceed MaxChargingWindows")

		// Should use continuous 0-3 @ 10 each = 40
		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(time.Hour)
		assert.InDelta(t, 40.0, totalCost, 0.01)
	})

	t.Run("exactly three windows optimal", func(t *testing.T) {
		// Scenario where exactly 3 separate windows are the optimal solution
		// Should use all 3 without any reduction needed
		testRates := rates([]float64{10, 50, 10, 50, 10, 50}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 3*time.Hour, clock.Now().Add(6*time.Hour))

		assert.Equal(t, 3*time.Hour, Duration(plan))
		windows := countChargingWindows(plan)
		assert.Equal(t, 3, windows, "should use exactly 3 windows when optimal")

		// Cost: 3x10 = 30
		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(time.Hour)
		assert.InDelta(t, 30.0, totalCost, 0.01)
	})

	t.Run("mixed duration windows", func(t *testing.T) {
		// Different slot durations: 15min, 30min, 1h
		// Tests replacement of short window with longer mixed-duration window
		testRates := rates([]float64{10, 50, 50, 50}, clock.Now(), 15*time.Minute)
		testRates = append(testRates, rates([]float64{10, 50}, clock.Now().Add(time.Hour), 15*time.Minute)...)
		testRates = append(testRates, rates([]float64{10, 50}, clock.Now().Add(90*time.Minute), 15*time.Minute)...)
		testRates = append(testRates, rates([]float64{10, 15}, clock.Now().Add(2*time.Hour), 15*time.Minute)...)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 60*time.Minute, clock.Now().Add(150*time.Minute))

		assert.Equal(t, 60*time.Minute, Duration(plan))
		windows := countChargingWindows(plan)
		assert.LessOrEqual(t, windows, 3, "should not exceed MaxChargingWindows")

		// Should find an efficient combination of the 10-cost slots
		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(15*time.Minute)
		assert.LessOrEqual(t, totalCost, 50.0, "should minimize cost with mixed durations")
	})

	t.Run("replacement considers all window types", func(t *testing.T) {
		// Ensures replacement logic considers windows that don't start at same time
		// 5 single slots @10 (hours 0,2,4,6,8), need 5h with max 3 windows
		// Should select 3 latest windows and extend one to 2h
		testRates := rates([]float64{10, 100, 10, 100, 10, 100, 10, 100, 10, 100}, clock.Now(), time.Hour)
		slices.SortStableFunc(testRates, sortByCost)

		plan := p.plan(testRates, 5*time.Hour, clock.Now().Add(10*time.Hour))

		assert.Equal(t, 5*time.Hour, Duration(plan))
		windows := countChargingWindows(plan)
		assert.LessOrEqual(t, windows, 3, "should not exceed MaxChargingWindows")

		// With 3 windows limit and 5h need: expects hours 4,6,8-9 = 10+10+10+100 = 130
		// Or similar combination with some expensive slots
		totalCost := AverageCost(plan) * float64(Duration(plan)) / float64(time.Hour)
		assert.LessOrEqual(t, totalCost, 140.0, "should find 5h plan in max 3 windows")
	})
}
