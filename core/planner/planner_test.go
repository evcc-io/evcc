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
)

func rates(prices []float64, start time.Time) api.Rates {
	res := make(api.Rates, 0, len(prices))

	for i, v := range prices {
		slotStart := start.Add(time.Duration(i) * time.Hour)
		ar := api.Rate{
			Start: slotStart,
			End:   slotStart.Add(1 * time.Hour),
			Price: v,
		}
		res = append(res, ar)
	}

	return res
}

func TestPlanner(t *testing.T) {
	dt := time.Hour
	ctrl := gomock.NewController(t)

	type se struct {
		delay     time.Duration
		cDuration time.Duration
		res       bool
	}

	tc := []struct {
		desc   string
		prices []float64
		end    time.Duration
		series []se
	}{
		{"falling prices", []float64{5, 4, 3, 2, 1, 0, 0, 0, 10}, 5 * time.Hour, []se{
			{1*dt - 1, 20 * time.Minute, false},
			{2*dt - 1, 20 * time.Minute, false},
			{3*dt - 1, 20 * time.Minute, false},
			{3 * dt, 20 * time.Minute, false},
			{4*dt - 1, 20 * time.Minute, false},
			{4*dt - 30*time.Minute, 20 * time.Minute, false}, // start as late as possible
			{5*dt - 20*time.Minute, 20 * time.Minute, true},
			{5 * dt, 5 * time.Minute, true}, // after desired charge timer,
		}},
		{"rising prices", []float64{1, 2, 3, 4, 5, 6, 7, 8}, 5 * time.Hour, []se{
			{1*dt - 1, time.Hour, true},
			{2*dt - 1, 5 * time.Minute, true}, // charging took longer than expected
			{3*dt - 1, 0, false},
			{5 * dt, 0, false}, // after desired charge timer
		}},
		{"last slot", []float64{5, 2, 5, 4, 3, 5, 5, 5, 10}, 5 * time.Hour, []se{
			{1*dt - 1, 70 * time.Minute, false},
			{2*dt - 1, 70 * time.Minute, true},
			{3*dt - 1, 20 * time.Minute, false},
			{4*dt - 1, 20 * time.Minute, false},
			{4 * dt, 20 * time.Minute, true}, // start as late as possible
			{4*dt + 40*time.Minute, 20 * time.Minute, true},
		}},
		{"don't pause last slot", []float64{5, 4, 5, 3, 2, 5, 5, 5, 10}, 5 * time.Hour, []se{
			{1*dt - 1, 70 * time.Minute, false},
			{2*dt - 1, 70 * time.Minute, false},
			{3*dt - 1, 70 * time.Minute, false},
			{4*dt - 1, 20 * time.Minute, false},
			{4 * dt, 20 * time.Minute, true}, // don't pause last slot
		}},
		{"delay expensive middle", []float64{5, 4, 3, 5, 5, 5, 5, 5, 10}, 5 * time.Hour, []se{
			{1*dt - 1, 70 * time.Minute, false},
			{1 * dt, 70 * time.Minute, false},
			{2*dt - 1, 61 * time.Minute, true}, // delayed start on expensive slot
			{3*dt - 1, 60 * time.Minute, true}, // cheapest slot
		}},
		{"fixed tariff", []float64{2}, 30 * time.Minute, []se{
			{1, 2 * time.Hour, true},
			{1, 10 * time.Minute, true},
		}},
		{"always expensive", []float64{5, 4, 3, 2, 1, 0, 0, 0, 10}, 5 * time.Hour, []se{
			{1*dt - 1, time.Minute, false},
			{2*dt - 1, time.Minute, false},
			{3*dt - 1, time.Minute, false},
			{4*dt - 1, time.Minute, false},
			{5*dt - 1, time.Minute, true}, // cheapest price
		}},
	}

	clck := clock.NewMock()

	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			t.Logf("set: %v", clck.Now())

			trf := mock.NewMockTariff(ctrl)
			trf.EXPECT().Rates().AnyTimes().Return(rates(tc.prices, clck.Now()), nil)

			p := &Planner{
				log:    util.NewLogger("foo"),
				clock:  clck,
				tariff: trf,
			}

			start := clck.Now()
			for idx, se := range tc.series {
				clck.Set(start.Add(se.delay))

				_, res, err := p.Active(se.cDuration, start.Add(tc.end))
				assert.NoError(t, err)
				assert.Equalf(t, se.res, res, "%s case %d: expected %v, got %v", tc.desc, idx+1, se.res, res)
			}
		})
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
	assert.True(t, res, "should start past target time")
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
	assert.True(t, res, "should start past target time")
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
	assert.True(t, res, "should plan if car can not be charged completely after known prices ")
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

	_, res, err := p.Active(time.Minute, clck.Now())
	assert.NoError(t, err)
	assert.True(t, res, "should start when target time reached and car not fully charged")

	_, res, err = p.Active(time.Hour, clck.Now().Add(-time.Hour))
	assert.NoError(t, err)
	assert.True(t, res, "should start when target time past and car not fully charged")

	_, res, err = p.Active(0, clck.Now().Add(-time.Hour))
	assert.NoError(t, err)
	assert.False(t, res, "should not start when fully charged")
}
