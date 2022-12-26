package planner

import (
	"sort"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
	clck := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clck.Now()), nil)

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clck,
	}

	rates, err := trf.Rates()
	assert.NoError(t, err)

	sort.Sort(rates)

	{
		plan := p.Plan(rates, time.Hour, clck.Now())
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
			clck.Now(),
			clck.Now().Add(6 * time.Hour),
			clck.Now().Add(2 * time.Hour),
			time.Hour,
			10,
		},
		{
			"plan 60-0-60-0-0-0",
			2 * time.Hour,
			clck.Now(),
			clck.Now().Add(6 * time.Hour),
			clck.Now().Add(0 * time.Hour),
			2 * time.Hour,
			30,
		},
		{
			"plan (30)30-0-60-0-0-0",
			time.Duration(90 * time.Minute),
			clck.Now(),
			clck.Now().Add(6 * time.Hour),
			clck.Now().Add(30 * time.Minute),
			time.Duration(90 * time.Minute),
			20,
		},

		{
			"plan 0-0-60-0-0-0",
			time.Hour,
			clck.Now().Add(30 * time.Minute),
			clck.Now().Add(6 * time.Hour),
			clck.Now().Add(2 * time.Hour),
			time.Hour,
			10,
		},
		{
			"plan (30)30-0-60-0-30(30)-0",
			2 * time.Hour,
			clck.Now().Add(30 * time.Minute),
			clck.Now().Add(6 * time.Hour),
			clck.Now().Add(30 * time.Minute),
			2 * time.Hour,
			40,
		},
		{
			"plan (30)30-0-60-0-0-0",
			time.Duration(90 * time.Minute),
			clck.Now().Add(30 * time.Minute),
			clck.Now().Add(6 * time.Hour),
			clck.Now().Add(30 * time.Minute),
			time.Duration(90 * time.Minute),
			20,
		},
	}

	for i, tc := range tc {
		t.Log(tc.desc)
		clck.Set(tc.now)
		plan := p.Plan(rates, tc.duration, tc.target)

		assert.Equalf(t, tc.planStart.UTC(), Start(plan).UTC(), "case %d start", i)
		assert.Equalf(t, tc.planDuration, Duration(plan), "case %d duration", i)
		assert.Equalf(t, tc.planCost, Cost(plan), "case %d cost", i)
	}
}

func TestNilTariff(t *testing.T) {
	clck := clock.NewMock()

	p := &Planner{
		log:   util.NewLogger("foo"),
		clock: clck,
	}

	_, res, err := p.Active(time.Hour, clck.Now().Add(30*time.Minute))
	assert.NoError(t, err)
	assert.True(t, res, "should start past start time")

	_, res, err = p.Active(time.Hour, clck.Now().Add(-30*time.Minute))
	assert.NoError(t, err)
	assert.False(t, res, "should not start past target time")
}

func TestFlatTariffTargetInThePast(t *testing.T) {
	clck := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clck.Now()), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clck,
		tariff: trf,
	}

	_, res, err := p.Active(time.Hour, clck.Now().Add(30*time.Minute))
	assert.NoError(t, err)
	assert.True(t, res, "should start past start time")

	_, res, err = p.Active(time.Hour, clck.Now().Add(-30*time.Minute))
	assert.NoError(t, err)
	assert.False(t, res, "should not start past target time")
}

func TestFlatTariffLongSlots(t *testing.T) {
	clck := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clck.Now(), func(start time.Time) time.Time {
		return start.Add(24 * time.Hour)
	}), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clck,
		tariff: trf,
	}

	_, res, err := p.Active(time.Hour, clck.Now().Add(2*time.Hour))
	assert.NoError(t, err)
	assert.False(t, res, "should not start long last slot before due time")

	_, res, err = p.Active(time.Hour, clck.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.True(t, res, "should start long last slot after due time")
}

func TestTargetAfterKnownPrices(t *testing.T) {
	clck := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0}, clck.Now()), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clck,
		tariff: trf,
	}

	_, res, err := p.Active(40*time.Minute, clck.Now().Add(2*time.Hour)) // charge efficiency does not allow to test with 1h
	assert.NoError(t, err)
	assert.False(t, res, "should not start if car can be charged completely after known prices ")

	_, res, err = p.Active(2*time.Hour, clck.Now().Add(2*time.Hour))
	assert.NoError(t, err)
	assert.True(t, res, "should start if car can not be charged completely after known prices ")
}

func TestChargeAfterTargetTime(t *testing.T) {
	clck := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := mock.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 0, 0, 0}, clck.Now()), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clck,
		tariff: trf,
	}

	_, res, err := p.Active(time.Hour, clck.Now())
	assert.NoError(t, err)
	assert.False(t, res, "should not start past target time")

	_, res, err = p.Active(time.Hour, clck.Now().Add(-time.Hour))
	assert.NoError(t, err)
	assert.False(t, res, "should not start past target time")
}
