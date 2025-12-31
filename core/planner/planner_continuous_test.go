package planner

import (
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

func TestContinuous_CheapestContiguousSlots(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)
	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.09},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.11},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.11},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.25},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	plan := p.Plan(2*time.Hour, 0, now.Add(6*time.Hour), true)

	require.Len(t, plan, 2)
	assert.Equal(t, rates[2].Start, plan[0].Start)
	assert.Equal(t, rates[3].End, plan[len(plan)-1].End)
	assert.Equal(t, rates[2].Value, plan[0].Value)
	assert.Equal(t, rates[3].Value, plan[1].Value)
}

func TestContinuous_WindowWithPastRates(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.05},
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.08},
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.07},
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.20},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.09},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.15},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.11},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.25},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true)

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)
	assert.False(t, plan[0].Start.Before(now))
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start)
	assert.Equal(t, now.Add(3*time.Hour), plan[len(plan)-1].End)
	assert.Equal(t, 0.09, plan[0].Value)
	assert.Equal(t, 0.10, plan[1].Value)
}

func TestContinuous_WindowAllRatesInPast(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now.Add(-6 * time.Hour), End: now.Add(-5 * time.Hour), Value: 0.05},
		{Start: now.Add(-5 * time.Hour), End: now.Add(-4 * time.Hour), Value: 0.08},
		{Start: now.Add(-4 * time.Hour), End: now.Add(-3 * time.Hour), Value: 0.07},
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.09},
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.10},
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.11},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(3 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	// When all rates are in the past and target is in future, expect nil plan
	assert.Empty(t, plan, "plan should be nil when all rates are in the past")
}

// TestContinuous_WindowRatesSpanningPastAndFuture tests continuous mode with rates
// spanning from past to future, where the optimal window would start in the past
func TestContinuous_WindowRatesSpanningPastAndFuture(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	// Rates spanning from 3h before now to 6h after now
	// The cheapest window would be -3h to -1h, but that's in the past
	rates := api.Rates{
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.05}, // cheapest, but past
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.06}, // cheap, but past
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.12},                     // partially past
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.15},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.08}, // cheapest future
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.09}, // second cheapest future
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.18},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.14},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.20},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)

	// Critical: plan must start at or after now, even if cheaper rates existed in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")

	// Should find cheapest 2-hour window starting from now or later
	// Expected: 1h-3h window (two slots with prices 0.08 and 0.09)
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start, "start should be at cheapest future window")
	assert.Equal(t, now.Add(3*time.Hour), plan[len(plan)-1].End, "end should match 2-hour window")
	assert.Equal(t, 0.08, plan[0].Value, "first slot should have actual price")
	assert.Equal(t, 0.09, plan[1].Value, "second slot should have actual price")
}

// TestContinuous_WindowRatesStartInFuture tests continuous mode when tariff data
// starts in the future, but target time is within the tariff data range
func TestContinuous_WindowRatesStartInFuture(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.08},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.09},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.15},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.18},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(5 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)

	// Plan must not start in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")

	// Should find cheapest 2-hour window within available rates
	// Expected: 2h-4h window (two slots with prices 0.08 and 0.09)
	assert.Equal(t, now.Add(2*time.Hour), plan[0].Start, "start should be at cheapest window in future rates")
	assert.Equal(t, now.Add(4*time.Hour), plan[len(plan)-1].End, "end should match 2-hour window")
	assert.Equal(t, 0.08, plan[0].Value, "first slot should have actual price")
	assert.Equal(t, 0.09, plan[1].Value, "second slot should have actual price")
}

func TestContinuous_WindowLateChargingPreference(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.10},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.10},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.10},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.10},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.15},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)

	// Should select the latest window with equal cost (3h-5h)
	// All windows from 0h-2h, 1h-3h, 2h-4h, and 3h-5h have the same total cost
	// But we prefer late charging, so 3h-5h should be selected
	assert.Equal(t, now.Add(3*time.Hour), plan[0].Start, "should select latest window with equal cost")
	assert.Equal(t, now.Add(5*time.Hour), plan[len(plan)-1].End, "end should be 2 hours after start")
	assert.Equal(t, 0.10, plan[0].Value, "first slot should have actual price")
	assert.Equal(t, 0.10, plan[1].Value, "second slot should have actual price")
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

	plan = p.Plan(tariff.SlotDuration, 24*time.Hour, clock.Now().Add(time.Hour), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now().Add(3 * tariff.SlotDuration),
			End:   clock.Now().Add(4 * tariff.SlotDuration),
			Value: 4,
		},
	}, plan, "all precondition")
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

func TestPrecondition_Everything(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	trf := api.NewMockTariff(ctrl)

	// Create 8 hours of rates with varying prices (cheaper toward the end)
	prices := []float64{10, 9, 8, 7, 6, 5, 4, 3}
	trf.EXPECT().Rates().AnyTimes().Return(rates(prices, clock.Now(), tariff.SlotDuration), nil)

	p := &Planner{
		log:    util.NewLogger("test"),
		clock:  clock,
		tariff: trf,
	}

	targetTime := clock.Now().Add(8 * tariff.SlotDuration) // 8 hours from now
	requiredDuration := 2 * tariff.SlotDuration            // need 2 hours
	precondition := 7 * 24 * time.Hour                     // "everything" = 7 days

	// Test with continuous=false (cheapest mode - should be ignored)
	plan := p.Plan(requiredDuration, precondition, targetTime, false)

	require.NotEmpty(t, plan, "plan should not be empty")

	// Plan should end exactly at target time
	assert.Equal(t, targetTime, plan[len(plan)-1].End, "plan should end at target time")

	// Plan should have total duration = requiredDuration (NOT precondition duration)
	totalDuration := Duration(plan)
	assert.Equal(t, requiredDuration, totalDuration, "plan duration should equal required duration, not precondition")

	// Plan should start at latest possible time (targetTime - requiredDuration)
	expectedStart := targetTime.Add(-requiredDuration)
	assert.Equal(t, expectedStart, plan[0].Start, "plan should start at latest possible time")

	// Should contain actual rate data (slots 6-7 with prices 4, 3)
	assert.Len(t, plan, 2, "should have 2 slots for 30-minute duration")
	assert.Equal(t, 4.0, plan[0].Value, "should have actual rate value from slot 6")
	assert.Equal(t, 3.0, plan[1].Value, "should have actual rate value from slot 7")

	// Test with continuous=true (should also be ignored when precondition=everything)
	planContinuous := p.Plan(requiredDuration, precondition, targetTime, true)
	assert.Equal(t, plan, planContinuous, "continuous flag should be ignored when precondition=everything")
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

// the target time (even at non-slot boundaries) by starting early
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
	requiredDuration := 2*time.Hour + 5*time.Minute // need 2h5m, have 3h10m available

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous, no precondition

	require.NotEmpty(t, plan)

	// Plan must not extend beyond target
	lastSlot := plan[len(plan)-1]
	assert.False(t, lastSlot.End.After(targetTime),
		"plan must not extend beyond target")

	// Total duration must equal required duration
	assert.Equal(t, requiredDuration, Duration(plan), "plan duration must match required")

	// Plan should use the cheapest slots (02:00-03:00 range, prices 0.08)
	avgCost := AverageCost(plan)
	assert.Less(t, avgCost, 0.12, "plan should use cheaper slots")

	// Target at 03:10 (non-slot boundary - must finish before target)
	requiredDurationShort := 12 * time.Minute                       // need 12m, have 3h10m available
	plan = planner.Plan(requiredDurationShort, 0, targetTime, true) // continuous, no precondition

	require.NotEmpty(t, plan)

	// Plan must not extend beyond target
	lastSlotShort := plan[len(plan)-1]
	assert.False(t, lastSlotShort.End.After(targetTime),
		"plan must not extend beyond target")

	// Total duration must equal required duration
	assert.Equal(t, requiredDurationShort, Duration(plan), "plan (short) duration must match required")

	// Plan should use the cheapest slots
	avgCostShort := AverageCost(plan)
	assert.Equal(t, 0.08, avgCostShort, "plan (short) should use cheapest slots (0.08)")

	// Target at 03:10 (non-slot boundary - must finish before target)
	requiredDurationMedium := 27 * time.Minute                       // need 27m, have 3h10m available
	plan = planner.Plan(requiredDurationMedium, 0, targetTime, true) // continuous, no precondition

	require.NotEmpty(t, plan)

	// Plan must not extend beyond target
	lastSlotMedium := plan[len(plan)-1]
	assert.False(t, lastSlotMedium.End.After(targetTime),
		"plan must not extend beyond target")

	// Total duration must equal required duration
	assert.Equal(t, requiredDurationMedium, Duration(plan), "plan (medium) duration must match required")

	// Plan should use the cheapest slots
	avgCostMedium := AverageCost(plan)
	assert.Equal(t, 0.08, avgCostMedium, "plan (medium) should use cheapest slots (0.08)")
}
