package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type MockTariff struct {
	api.Tariff
	start  time.Time
	prices []float64
}

func (m MockTariff) Rates() ([]api.Rate, error) {
	start := m.start

	var res []api.Rate
	for _, v := range m.prices {
		ar := api.Rate{
			Start: start,
			End:   start.Add(1 * time.Hour),
			Price: v,
		}
		start = ar.End
		res = append(res, ar)
	}

	return res, nil
}

func TestIsCheapSlotNow(t *testing.T) {
	type se struct {
		caseNr    int
		delay     time.Duration
		cDuration time.Duration
		res       bool
	}
	dt := time.Hour
	tc := []struct {
		desc   string
		prices []float64
		end    time.Duration
		series []se
	}{
		{"falling prices", []float64{5, 4, 3, 2, 1, 0, 0, 0}, 5 * time.Hour, []se{
			{1, 1*dt - 1, 20 * time.Minute, false},
			{2, 2*dt - 1, 20 * time.Minute, false},
			{3, 3*dt - 1, 20 * time.Minute, false},
			{4, 3*dt + 1, 20 * time.Minute, false},
			{5, 4*dt - 1, 20 * time.Minute, false},
			{6, 4*dt - 30*time.Minute, 20 * time.Minute, false}, // start as late as possible
			{7, 5*dt - 20*time.Minute, 20 * time.Minute, true},
			{8, 5*dt + 1, 5 * time.Minute, false}, // after desired charge timer,
		}},
		{"raising prices", []float64{1, 2, 3, 4, 5, 6, 7, 8}, 5 * time.Hour, []se{
			{1, 1*dt - 1, time.Hour, true},
			{2, 2*dt - 1, 5 * time.Minute, true}, // charging took longer than expected
			{3, 3*dt - 1, 0, false},
			{4, 5*dt + 1, 0, false}, // after desired charge timer
		}},
		{"last slot", []float64{5, 2, 5, 4, 3, 5, 5, 5}, 5 * time.Hour, []se{
			{1, 1*dt - 1, 70 * time.Minute, false},
			{2, 2*dt - 1, 70 * time.Minute, true},
			{3, 3*dt - 1, 20 * time.Minute, false},
			{4, 4*dt - 1, 20 * time.Minute, false},
			{5, 4*dt + 1, 20 * time.Minute, true}, // start as late as possible
			{6, 4*dt + 40*time.Minute, 20 * time.Minute, true},
		}},
		{"don't stop for last slot", []float64{5, 4, 5, 2, 3, 5, 5, 5}, 5 * time.Hour, []se{
			{1, 1*dt - 1, 70 * time.Minute, false},
			{2, 2*dt - 1, 70 * time.Minute, false},
			{3, 3*dt - 1, 70 * time.Minute, false},
			{4, 4*dt - 1, 20 * time.Minute, true}, // don't pause last slot
			{5, 4*dt + 1, 20 * time.Minute, true},
		}},
		{"delay expensiv middle", []float64{5, 4, 3, 5, 5, 5, 5, 5}, 5 * time.Hour, []se{
			{1, 1*dt - 1, 70 * time.Minute, false},
			{1, 1*dt + 1, 70 * time.Minute, false},
			{2, 2*dt - 1, 61 * time.Minute, true}, // delayed start on expensiv slot
			{3, 3*dt - 1, 60 * time.Minute, true}, // cheapest slot
		}},
		{"disable after known prices, 1h", []float64{5, 4, 3, 2, 1, 0, 0, 0}, 5 * time.Hour, []se{
			{1, 20 * dt, time.Hour, false},
		}},
		{"fixed tariff", []float64{2}, 5 * time.Hour, []se{
			{1, 0, 2 * time.Hour, true},
			{1, 0, 10 * time.Minute, true},
		}},
	}

	clck := clock.NewMock()
	start := clck.Now()
	for _, tc := range tc {
		t.Logf("%+v", tc.desc)
		clck.Set(start)

		p := &Planner{
			log:    util.NewLogger("foo"),
			clock:  clck,
			tariff: MockTariff{prices: tc.prices, start: start},
		}

		for _, se := range tc.series {
			clck.Set(start.Add(se.delay))

			if res, _ := p.PlanActive(se.cDuration, start.Add(tc.end)); se.res != res {
				t.Errorf("%s case %v: expected %v, got %v", tc.desc, se.caseNr, se.res, res)
			}
		}
	}
}

func TestIsCheap(t *testing.T) {
	type se struct {
		caseNr    int
		delay     time.Duration
		cDuration time.Duration
		res       bool
	}
	dt := time.Hour
	tc := []struct {
		desc   string
		prices []float64
		end    time.Duration
		series []se
	}{
		{"always expensive", []float64{5, 4, 3, 2, 1, 0, 0, 0}, 5 * time.Hour, []se{
			{1, 1*dt - 1, time.Minute, false},
			{2, 2*dt - 1, time.Minute, false},
			{3, 3*dt - 1, time.Minute, false},
			{4, 4*dt - 1, time.Minute, false},
			{5, 5*dt - 1, time.Minute, true}, // cheapest price
		}},
	}

	clck := clock.NewMock()
	start := clck.Now()
	for _, tc := range tc {
		t.Logf("%+v", tc.desc)
		clck.Set(start)

		p := &Planner{
			log:    util.NewLogger("foo"),
			clock:  clck,
			tariff: MockTariff{prices: tc.prices, start: start},
		}

		for _, se := range tc.series {
			clck.Set(start.Add(se.delay))

			if res, _ := p.PlanActive(se.cDuration, start.Add(tc.end)); se.res != res {
				t.Errorf("%s case %v: expected %v, got %v", tc.desc, se.caseNr, se.res, res)
			}
		}
	}
}
