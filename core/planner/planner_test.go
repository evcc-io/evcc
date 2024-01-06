package planner

import (
	"slices"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rates(prices []float64, start time.Time, slotDuration time.Duration) api.Rates {
	res := make(api.Rates, 0, len(prices))

	for i, v := range prices {
		slotStart := start.Add(time.Duration(i) * time.Hour)
		ar := api.Rate{
			Start: slotStart,
			End:   slotStart.Add(slotDuration),
			Price: v,
		}
		res = append(res, ar)
	}

	return res
}

// TODO start before start of rates

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

func TestNilTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan, err := p.Plan(time.Hour, clock.Now().Add(30*time.Minute))
	require.NoError(t, err)
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

	plan, err := p.Plan(time.Hour, clock.Now().Add(30*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan, err = p.Plan(time.Hour, clock.Now().Add(-30*time.Minute))
	require.NoError(t, err)
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
	plan, err := p.Plan(time.Hour, clock.Now().Add(2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}, SlotAt(clock.Now(), plan))
	assert.Equal(t, api.Rate{}, SlotAt(clock.Now().Add(time.Hour), plan))

	// expect 00:00-01:00 UTC
	plan, err = p.Plan(time.Hour, clock.Now().Add(time.Hour))
	require.NoError(t, err)
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

	plan, err := p.Plan(40*time.Minute, clock.Now().Add(2*time.Hour)) // charge efficiency does not allow to test with 1h
	require.NoError(t, err)
	assert.False(t, !SlotAt(clock.Now(), plan).IsEmpty(), "should not start if car can be charged completely after known prices ")

	plan, err = p.Plan(2*time.Hour, clock.Now().Add(2*time.Hour))
	require.NoError(t, err)
	assert.True(t, !SlotAt(clock.Now(), plan).IsEmpty(), "should start if car can not be charged completely after known prices ")
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

	plan, err := p.Plan(time.Hour, clock.Now())
	require.NoError(t, err)
	assert.Equal(t, simplePlan, plan, "expected simple plan")

	plan, err = p.Plan(time.Hour, clock.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, simplePlan, plan, "expected simple plan")
}

func TestContinuousPlanNoTariff(t *testing.T) {
	clock := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	plan, err := p.Plan(time.Hour, clock.Now())
	require.NoError(t, err)

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

	plan, err := p.Plan(150*time.Minute, clock.Now())
	require.NoError(t, err)

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

	plan, err := p.Plan(30*time.Minute, clock.Now())
	require.NoError(t, err)

	// 3-slot plan
	assert.Len(t, plan, 1)
}
