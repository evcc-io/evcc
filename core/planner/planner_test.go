package planner

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func rates(prices []float64, start time.Time, slotEndFunc ...func(time.Time) time.Time) api.Rates {
	res := make(api.Rates, 0, len(prices))

	slotEnd := func(start time.Time) time.Time {
		return start.Add(time.Hour)
	}

	if len(slotEndFunc) == 1 {
		slotEnd = slotEndFunc[0]
	}

	for i, v := range prices {
		slotStart := start.Add(time.Duration(i) * time.Hour)
		ar := api.Rate{
			Start: slotStart,
			End:   slotEnd(slotStart),
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

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now()), nil)

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clock,
	}

	rates, err := trf.Rates()
	assert.NoError(t, err)

	slices.SortStableFunc(rates, sortByCost)

	{
		plan := p.plan(rates, time.Hour, clock.Now())
		assert.Equal(t, 0, len(plan))
	}

	tc := []struct {
		desc string
		// task
		duration time.Duration
		now      time.Time
		target   time.Time
		// result
		planStart    time.Time
		planDuration time.Duration
		planCost     float64
	}{
		// numbers in brackets denote inactive partial slots
		{
			"plan 0-0-60-0-0-0",
			time.Hour,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(2 * time.Hour),
			time.Hour,
			10,
		},
		{
			"plan 60-0-60-0-0-0",
			2 * time.Hour,
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(0 * time.Hour),
			2 * time.Hour,
			30,
		},
		{
			"plan (30)30-0-60-0-0-0",
			time.Duration(90 * time.Minute),
			clock.Now(),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			time.Duration(90 * time.Minute),
			20,
		},

		{
			"plan 0-0-60-0-0-0",
			time.Hour,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(2 * time.Hour),
			time.Hour,
			10,
		},
		{
			"plan (30)30-0-60-0-30(30)-0",
			2 * time.Hour,
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			2 * time.Hour,
			40,
		},
		{
			"plan (30)30-0-60-0-0-0",
			time.Duration(90 * time.Minute),
			clock.Now().Add(30 * time.Minute),
			clock.Now().Add(6 * time.Hour),
			clock.Now().Add(30 * time.Minute),
			time.Duration(90 * time.Minute),
			20,
		},
	}

	for i, tc := range tc {
		t.Log(tc.desc)
		clock.Set(tc.now)
		plan := p.plan(rates, tc.duration, tc.target)

		assert.Equalf(t, tc.planStart.UTC(), Start(plan).UTC(), "case %d start", i)
		assert.Equalf(t, tc.planDuration, Duration(plan), "case %d duration", i)
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
	assert.NoError(t, err)
	assert.True(t, !ActiveSlot(clock, plan).End.IsZero(), "should start past start time")

	plan, err = p.Plan(time.Hour, clock.Now().Add(-30*time.Minute))
	assert.NoError(t, err)
	assert.False(t, !ActiveSlot(clock, plan).End.IsZero(), "should not start past target time")
}

func TestFlatTariffTargetInThePast(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now()), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan, err := p.Plan(time.Hour, clock.Now().Add(30*time.Minute))
	assert.NoError(t, err)
	assert.True(t, !ActiveSlot(clock, plan).End.IsZero(), "should start past start time")

	plan, err = p.Plan(time.Hour, clock.Now().Add(-30*time.Minute))
	assert.NoError(t, err)
	assert.False(t, !ActiveSlot(clock, plan).End.IsZero(), "should not start past target time")
}

func TestFlatTariffLongSlots(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now(), func(start time.Time) time.Time {
		return start.Add(24 * time.Hour)
	}), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan, err := p.Plan(time.Hour, clock.Now().Add(2*time.Hour))
	assert.NoError(t, err)
	assert.False(t, !ActiveSlot(clock, plan).End.IsZero(), "should not start long last slot before due time")

	plan, err = p.Plan(time.Hour, clock.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.True(t, !ActiveSlot(clock, plan).End.IsZero(), "should start long last slot after due time")
}

func TestTargetAfterKnownPrices(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clock.Now()), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan, err := p.Plan(40*time.Minute, clock.Now().Add(2*time.Hour)) // charge efficiency does not allow to test with 1h
	assert.NoError(t, err)
	assert.False(t, !ActiveSlot(clock, plan).End.IsZero(), "should not start if car can be charged completely after known prices ")

	plan, err = p.Plan(2*time.Hour, clock.Now().Add(2*time.Hour))
	assert.NoError(t, err)
	assert.True(t, !ActiveSlot(clock, plan).End.IsZero(), "should start if car can not be charged completely after known prices ")
}

func TestChargeAfterTargetTime(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 0, 0, 0}, clock.Now()), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	plan, err := p.Plan(time.Hour, clock.Now())
	assert.NoError(t, err)
	assert.False(t, !ActiveSlot(clock, plan).End.IsZero(), "should not start past target time")

	plan, err = p.Plan(time.Hour, clock.Now().Add(-time.Hour))
	assert.NoError(t, err)
	assert.False(t, !ActiveSlot(clock, plan).End.IsZero(), "should not start past target time")
}
