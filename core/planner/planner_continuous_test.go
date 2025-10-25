package planner

import (
	"slices"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestContinuous_Plan(t *testing.T) {
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
		// numbers in brackets denote inactive partial slots
		{
			"plan 0-0-60-0-0-0",
			time.Hour,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(2 * time.Hour),
			10,
		},
		{
			"plan 60-0-60-0-0-0",
			2 * time.Hour,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(0 * time.Hour),
			30,
		},
		{
			"plan (30)30-0-60-0-0-0",
			90 * time.Minute,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			20,
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
			"plan (30)30-0-60-0-30(30)-0",
			2 * time.Hour,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			40,
		},
		{
			"plan (30)30-0-60-0-0-0",
			90 * time.Minute,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			20,
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

func TestContinuous_NilTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(60 * time.Minute),
		},
	}, plan, "expected simple plan")
}

func TestContinuous_RatesError(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(nil, api.ErrOutdated)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(60 * time.Minute),
		},
	}, plan, "expected simple plan")
}

func TestContinuous_FlatTariffTargetInThePast(t *testing.T) {
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

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute), true)
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan = p.Plan(time.Hour, 0, clock.Now().Add(-30*time.Minute), true)
	assert.Equal(t, simplePlan, plan, "expected simple plan")
}

func TestContinuous_FlatTariffLongSlots(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now(), 24*time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// for a single slot, we always expect charging to start early because tariffs ensure
	// that slots are not longer than 1 hour and with that context this is not a problem

	// expect 00:00-01:00 UTC
	plan := p.Plan(time.Hour, 0, clock.Now().Add(2*time.Hour), true)
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}, SlotAt(clock.Now(), plan))
	assert.Equal(t, api.Rate{}, SlotAt(clock.Now().Add(time.Hour), plan))

	// expect 00:00-01:00 UTC
	plan = p.Plan(time.Hour, 0, clock.Now().Add(time.Hour), true)
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}, SlotAt(clock.Now(), plan))
}

func TestContinuous_TargetAfterKnownPrices(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(40*time.Minute, 0, clock.Now().Add(2*time.Hour), true) // charge efficiency does not allow to test with 1h
	assert.False(t, !SlotAt(clock.Now(), plan).IsZero(), "should not start if car can be charged completely after known prices ")

	plan = p.Plan(2*time.Hour, 0, clock.Now().Add(2*time.Hour), true)
	assert.True(t, !SlotAt(clock.Now(), plan).IsZero(), "should start if car can not be charged completely after known prices ")
}

func TestContinuous_ChargeAfterTargetTime(t *testing.T) {
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

	plan := p.Plan(time.Hour, 0, clock.Now(), true)
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan = p.Plan(time.Hour, 0, clock.Now().Add(-time.Hour), true)
	assert.Equal(t, simplePlan, plan, "expected simple plan")
}

func TestContinuous_Precondition(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	trf := api.NewMockTariff(ctrl)

	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{1, 2, 3, 4}, clock.Now(), tariff.SlotDuration), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(tariff.SlotDuration, tariff.SlotDuration, clock.Now().Add(4*tariff.SlotDuration), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now().Add(3 * tariff.SlotDuration),
			End:   clock.Now().Add(4 * tariff.SlotDuration),
			Value: 4,
		},
	}, plan, "expected last slot")

	plan = p.Plan(2*tariff.SlotDuration, tariff.SlotDuration, clock.Now().Add(4*tariff.SlotDuration), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(1 * tariff.SlotDuration),
			Value: 1,
		},
		{
			Start: clock.Now().Add(3 * tariff.SlotDuration),
			End:   clock.Now().Add(4 * tariff.SlotDuration),
			Value: 4,
		},
	}, plan, "expected two slots")

	plan = p.Plan(time.Duration(1.5*float64(tariff.SlotDuration)), tariff.SlotDuration, clock.Now().Add(4*tariff.SlotDuration), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(time.Duration(0.5 * float64(tariff.SlotDuration))),
			Value: 1,
		},
		{
			Start: clock.Now().Add(3 * tariff.SlotDuration),
			End:   clock.Now().Add(4 * tariff.SlotDuration),
			Value: 4,
		},
	}, plan, "expected trimmed slot at beginning and precondition slot")
}

func TestContinuous_Precondition_NonSlotBoundary(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	trf := api.NewMockTariff(ctrl)

	slotDuration := 15 * time.Minute

	// Create rates with 15-minute slots covering 8 hours (32 slots)
	prices := make([]float64, 32)
	for i := range prices {
		prices[i] = float64(i + 1)
	}
	trf.EXPECT().Rates().AnyTimes().Return(rates(prices, clock.Now(), slotDuration), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// Target time at 7:20 (non-slot boundary, between 7:15 and 7:30)
	// 7:20 is 29 slots + 5 minutes from now
	targetTime := clock.Now().Add(29*slotDuration + 5*time.Minute)

	// 30 minutes preconditioning, 1 hour charging
	precondition := 30 * time.Minute
	requiredDuration := 1 * time.Hour

	plan := p.Plan(requiredDuration, precondition, targetTime, true)

	// Verify precondition ends exactly at target time
	require.NotEmpty(t, plan)
	lastSlot := plan[len(plan)-1]
	assert.Equal(t, targetTime, lastSlot.End, "precondition should end exactly at target time")

	// Calculate total precondition duration
	var precondDuration time.Duration
	// Precondition starts at targetTime - 30min = 6:50
	precondStart := targetTime.Add(-precondition)
	for _, slot := range plan {
		if !slot.Start.Before(precondStart) {
			precondDuration += slot.End.Sub(slot.Start)
		}
	}
	assert.Equal(t, precondition, precondDuration, "total precondition duration should be exactly 30 minutes")

	// In continuous mode, find cheapest continuous 30min window (after precondition reduction)
	// Cheapest window: 01:00-01:30 (slots 0-1, prices 1+2)
	// Precondition: 07:50-08:20 (exactly 30min before target at 08:20)
	expectedPlan := api.Rates{
		// Charging slots (cheapest continuous 30 minutes)
		{Start: clock.Now(), End: clock.Now().Add(slotDuration), Value: 1},
		{Start: clock.Now().Add(slotDuration), End: clock.Now().Add(2 * slotDuration), Value: 2},
		// Precondition slots (exactly 30min before target, trimmed at both ends)
		{Start: targetTime.Add(-precondition), End: clock.Now().Add(28 * slotDuration), Value: 28},
		{Start: clock.Now().Add(28 * slotDuration), End: clock.Now().Add(29 * slotDuration), Value: 29},
		{Start: clock.Now().Add(29 * slotDuration), End: targetTime, Value: 30},
	}

	assert.Equal(t, expectedPlan, plan, "expected continuous charging slots and trimmed precondition slots")
}

func TestContinuous_ContinuousPlanNoTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan := p.Plan(time.Hour, 0, clock.Now(), true)

	// single-slot plan
	assert.Len(t, plan, 1)
	assert.Equal(t, clock.Now(), SlotAt(clock.Now(), plan).Start)
	assert.Equal(t, clock.Now().Add(time.Hour), SlotAt(clock.Now(), plan).End)
}

func TestContinuous_ContinuousPlan(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now().Add(time.Hour), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(150*time.Minute, 0, clock.Now(), true)

	// 3-slot plan
	assert.Len(t, plan, 3)
}

func TestContinuous_ContinuousPlanOutsideRates(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now().Add(time.Hour), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(30*time.Minute, 0, clock.Now(), true)

	// 3-slot plan
	assert.Len(t, plan, 1)
}

// TestContinuous_StartBeforeRates tests that when current time is before
// the first available rate, the planner waits and starts charging when
// rates become available, as long as there's enough time to reach the target
func TestContinuous_StartBeforeRates(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)
	log := util.NewLogger("test")

	// Rates start 2 hours in the future (gap from now until first rate)
	rates := api.Rates{
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.15},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.08}, // cheapest
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.20},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan, "plan should not be empty")
	require.Len(t, plan, 1, "should create single slot with actual price")

	// Should wait until rates are available and pick the cheapest slot
	assert.Equal(t, now.Add(4*time.Hour), plan[0].Start, "should start at cheapest available rate")
	assert.Equal(t, now.Add(5*time.Hour), plan[0].End, "should end after required duration")
	assert.Equal(t, 0.08, plan[0].Value, "should have actual price from cheapest slot")

	// Plan must not start before rates are available
	assert.False(t, plan[0].Start.Before(rates[0].Start), "plan must not start before first available rate")
}

// TestContinuous_StartBeforeRatesInsufficientTime tests that when current time
// is before the first available rate AND there's not enough time after rates
// start to complete charging before target, the planner starts charging as soon
// as rates become available (best effort approach)
func TestContinuous_StartBeforeRatesInsufficientTime(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)
	log := util.NewLogger("test")

	// Rates start 2 hours in the future, but we need 3 hours to charge
	// and target is only 4 hours away (not enough time to fully charge)
	rates := api.Rates{
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.15},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(4 * time.Hour)
	requiredDuration := 3 * time.Hour // Need 3h but only 2h available after rates start

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan, "plan should not be empty")

	// Best effort: start immediately to maximize charging time
	assert.Equal(t, now, plan[0].Start, "should start immediately")
	assert.Equal(t, 0.0, plan[0].Value, "gap-filling slot before rates has no price")
}

// TestContinuous_StartBeforeRatesSufficientTime tests that when current time
// is before the first available rate AND there IS enough time to complete
// charging, the planner finds the cheapest continuous window
func TestContinuous_StartBeforeRatesSufficientTime(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)
	log := util.NewLogger("test")

	// Rates start 2 hours in the future, we need 2 hours to charge
	// and target is 8 hours away (enough time to optimize)
	rates := api.Rates{
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.20},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.15},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.10}, // cheapest
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.08}, // cheapest
		{Start: now.Add(6 * time.Hour), End: now.Add(7 * time.Hour), Value: 0.12},
		{Start: now.Add(7 * time.Hour), End: now.Add(8 * time.Hour), Value: 0.25},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(8 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan, "plan should not be empty")
	require.Len(t, plan, 2, "should find 2-hour continuous window")

	// Should find cheapest continuous 2-hour window (04:00-06:00)
	assert.Equal(t, now.Add(4*time.Hour), plan[0].Start, "should start at cheapest window")
	assert.Equal(t, 0.10, plan[0].Value, "first slot should have cheapest window price")
	assert.Equal(t, 0.08, plan[1].Value, "second slot should have cheapest window price")
}

// TestContinuous_ExcessTimeFinishesAtTarget tests that with continuous mode,
// no precondition, and excess time available, the plan finishes exactly at
// the target time (even at non-slot boundaries) by starting early and using excess time
func TestContinuous_ExcessTimeFinishesAtTarget(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)
	log := util.NewLogger("test")

	slotDuration := 15 * time.Minute

	// Create 20 slots of 15 minutes each (5 hours total)
	// Prices: cheaper in the middle slots
	prices := []float64{
		0.30, 0.30, 0.30, 0.30, // 00:00-01:00 expensive
		0.15, 0.10, 0.10, 0.10, // 01:00-02:00 medium+cheap
		0.08, 0.08, 0.08, 0.08, // 02:00-03:00 cheapest
		0.12, 0.12, 0.12, 0.12, // 03:00-04:00 medium
		0.20, 0.20, 0.20, 0.20, // 04:00-05:00 expensive
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates(prices, now, slotDuration), nil)

	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: trf,
	}

	// Target at 03:10 (non-slot boundary - 10 minutes into the 03:00-03:15 slot)
	targetTime := now.Add(3*time.Hour + 10*time.Minute)
	requiredDuration := 2 * time.Hour // need 2h, have 3h10m available

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous, no precondition

	require.NotEmpty(t, plan)

	// Debug: print plan slots
	for i, slot := range plan {
		t.Logf("slot %d: %v - %v (duration: %v, price: %.2f)", i, slot.Start.Format("15:04"), slot.End.Format("15:04"), slot.End.Sub(slot.Start), slot.Value)
	}
	t.Logf("total plan duration: %v", Duration(plan))

	// CRITICAL: Plan must finish EXACTLY at target time (03:10)
	lastSlot := plan[len(plan)-1]
	assert.Equal(t, targetTime, lastSlot.End, "plan must finish exactly at target time 03:10")

	// Total duration must equal required duration
	assert.Equal(t, requiredDuration, Duration(plan), "plan duration must match required")

	// Should pick cheapest continuous 2h window ending at target: 01:10-03:10 (using cheapest slots)
	// First slot should start at 01:10 (using excess time to start early)
	assert.Equal(t, now.Add(time.Hour+10*time.Minute), plan[0].Start, "should start at 01:10 to finish at target")
}
