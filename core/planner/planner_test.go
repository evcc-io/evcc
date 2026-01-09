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

func TestClampRates(t *testing.T) {
	clock := clock.NewMock()
	rr := rates([]float64{0, 1}, clock.Now(), time.Hour)

	assert.Equal(t, rr, clampRates(rr, clock.Now(), clock.Now().Add(2*time.Hour)))
	assert.Equal(t, rates([]float64{0}, clock.Now(), time.Hour), clampRates(rr, clock.Now(), clock.Now().Add(time.Hour)))

	exp := api.Rates{{Start: clock.Now().Add(time.Hour), End: clock.Now().Add(2 * time.Hour), Value: 1}}
	assert.Equal(t, exp, clampRates(rr, clock.Now().Add(time.Hour), clock.Now().Add(2*time.Hour)))
}

func TestPlan(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	rates, err := trf.Rates()
	require.NoError(t, err)

	slices.SortStableFunc(rates, sortByCost)

	{
		// filter rates to [now, now] window - should return empty
		plan := optimalPlan(clampRates(rates, clock.Now(), clock.Now()), time.Hour, clock.Now())
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
		// filter rates to [now, target] window as caller would do
		plan := optimalPlan(clampRates(rates, tc.now, tc.target), tc.duration, tc.target)

		assert.Equalf(t, tc.planStart.UTC(), Start(plan).UTC(), "case %d start", i)
		assert.Equalf(t, tc.duration, Duration(plan), "case %d duration", i)
		assert.Equalf(t, tc.planCost, AverageCost(plan)*float64(Duration(plan))/float64(time.Hour), "case %d cost", i)
	}
}

func TestNilTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute), false)
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

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute), false)
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

	plan := p.Plan(time.Hour, 0, clock.Now().Add(30*time.Minute), false)
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan = p.Plan(time.Hour, 0, clock.Now().Add(-30*time.Minute), false)
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

	// for a single slot, we always expect charging to start early because tariffs ensure
	// that slots are not longer than 1 hour and with that context this is not a problem

	// expect 00:00-01:00 UTC
	plan := p.Plan(time.Hour, 0, clock.Now().Add(2*time.Hour), false)
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}, SlotAt(clock.Now(), plan))
	assert.Equal(t, api.Rate{}, SlotAt(clock.Now().Add(time.Hour), plan))

	// expect 00:00-01:00 UTC
	plan = p.Plan(time.Hour, 0, clock.Now().Add(time.Hour), false)
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

	plan := p.Plan(40*time.Minute, 0, clock.Now().Add(2*time.Hour), false) // charge efficiency does not allow to test with 1h
	assert.False(t, !SlotAt(clock.Now(), plan).IsZero(), "should not start if car can be charged completely after known prices ")

	plan = p.Plan(2*time.Hour, 0, clock.Now().Add(2*time.Hour), false)
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

	plan := p.Plan(time.Hour, 0, clock.Now(), false)
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan = p.Plan(time.Hour, 0, clock.Now().Add(-time.Hour), false)
	assert.Equal(t, simplePlan, plan, "expected simple plan")
}

func TestPrecondition(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{1, 2, 3, 4}, clock.Now(), tariff.SlotDuration), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(tariff.SlotDuration, tariff.SlotDuration, clock.Now().Add(4*tariff.SlotDuration), false)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now().Add(3 * tariff.SlotDuration),
			End:   clock.Now().Add(4 * tariff.SlotDuration),
			Value: 4,
		},
	}, plan, "expected last slot")

	plan = p.Plan(2*tariff.SlotDuration, tariff.SlotDuration, clock.Now().Add(4*tariff.SlotDuration), false)

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

	plan = p.Plan(time.Duration(1.5*float64(tariff.SlotDuration)), tariff.SlotDuration, clock.Now().Add(4*tariff.SlotDuration), false)
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

	plan = p.Plan(tariff.SlotDuration, 24*time.Hour, clock.Now().Add(time.Hour), false)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now().Add(3 * tariff.SlotDuration),
			End:   clock.Now().Add(4 * tariff.SlotDuration),
			Value: 4,
		},
	}, plan, "all precondition")
}

func TestPrecondition_NonSlotBoundary(t *testing.T) {
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

	plan := p.Plan(requiredDuration, precondition, targetTime, false)

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

	// Verify expected slots structure
	// Note: precondition (30min) reduces effective required duration from 1h to 30min
	// Cheapest 30min charging: slots at 01:00-01:30 (slots 0-1, prices 1,2)
	// Precondition: 07:50-08:20 (exactly 30min before target at 08:20)
	//   - 07:45-08:00 (slot 27, price 28) -> trimmed to 07:50-08:00 (10min)
	//   - 08:00-08:15 (slot 28, price 29) -> full slot (15min)
	//   - 08:15-08:30 (slot 29, price 30) -> trimmed to 08:15-08:20 (5min)
	expectedPlan := api.Rates{
		// Charging slots (cheapest 30 minutes after precondition reduction)
		{Start: clock.Now(), End: clock.Now().Add(slotDuration), Value: 1},
		{Start: clock.Now().Add(slotDuration), End: clock.Now().Add(2 * slotDuration), Value: 2},
		// Precondition slots (exactly 30min before target, trimmed at both ends)
		{Start: targetTime.Add(-precondition), End: clock.Now().Add(28 * slotDuration), Value: 28},
		{Start: clock.Now().Add(28 * slotDuration), End: clock.Now().Add(29 * slotDuration), Value: 29},
		{Start: clock.Now().Add(29 * slotDuration), End: targetTime, Value: 30},
	}

	assert.Equal(t, expectedPlan, plan, "expected charging slots and trimmed precondition slots")
}

func TestContinuousPlanNoTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan := p.Plan(time.Hour, 0, clock.Now(), false)

	// single-slot plan
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

	plan := p.Plan(150*time.Minute, 0, clock.Now(), false)

	// 3-slot plan
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

	plan := p.Plan(30*time.Minute, 0, clock.Now(), false)

	// 3-slot plan
	assert.Len(t, plan, 1)
}

// TestStartBeforeRates tests that when current time is before
// the first available rate, the planner waits and starts charging when
// rates become available, as long as there's enough time to reach the target
func TestStartBeforeRates(t *testing.T) {
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

// TestStartBeforeRatesInsufficientTime tests that when current time
// is before the first available rate AND there's not enough rate coverage
// to complete charging, the planner falls back to simplePlan (ignoring rates)
func TestStartBeforeRatesInsufficientTime(t *testing.T) {
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
	requiredDuration := 3 * time.Hour // Need 3h but only 2h rate coverage

	plan := planner.Plan(requiredDuration, 0, targetTime, false) // dispersed mode

	require.NotEmpty(t, plan, "plan should not be empty")

	// Insufficient rate coverage: fall back to simplePlan starting at latestStart
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start, "should start at latestStart (target - required)")
	assert.Equal(t, 0.0, plan[0].Value, "simplePlan has no price info")
}

// TestEmptyRatesAfterClamping tests fallback to simplePlan when no rates cover [now, targetTime]
func TestEmptyRatesAfterClamping(t *testing.T) {
	c := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0.20}, c.Now().Add(3*time.Hour), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("test"),
		clock:  c,
		tariff: trf,
	}

	plan := p.Plan(time.Hour, 0, c.Now().Add(90*time.Minute), false)
	require.Len(t, plan, 1)
	assert.Equal(t, c.Now().Add(30*time.Minute), plan[0].Start)
}
