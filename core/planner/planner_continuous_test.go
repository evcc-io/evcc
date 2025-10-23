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
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 1, 2, 3}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan := p.Plan(time.Hour, time.Hour, clock.Now().Add(4*time.Hour), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now().Add(3 * time.Hour),
			End:   clock.Now().Add(4 * time.Hour),
			Value: 3,
		},
	}, plan, "expected last slot")

	plan = p.Plan(2*time.Hour, time.Hour, clock.Now().Add(4*time.Hour), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now(),
			End:   clock.Now().Add(1 * time.Hour),
			Value: 0,
		},
		{
			Start: clock.Now().Add(3 * time.Hour),
			End:   clock.Now().Add(4 * time.Hour),
			Value: 3,
		},
	}, plan, "expected two slots")

	plan = p.Plan(time.Hour, 30*time.Minute, clock.Now().Add(4*time.Hour), true)
	assert.Equal(t, api.Rates{
		{
			Start: clock.Now().Add(30 * time.Minute),
			End:   clock.Now().Add(time.Hour),
			Value: 0,
		},
		{
			Start: clock.Now().Add(210 * time.Minute),
			End:   clock.Now().Add(4 * time.Hour),
			Value: 3,
		},
	}, plan, "expected short early and split late slot")
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

	require.NotEmpty(t, plan, "plan should not be empty - starts when rates become available")
	
	// Best effort: start as soon as rates are available
	assert.Equal(t, now.Add(2*time.Hour), plan[0].Start, "should start at first available rate")
	assert.Equal(t, 0.10, plan[0].Value, "should use first available rate price")
}