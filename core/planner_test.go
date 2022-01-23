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
	start        time.Time
	feature      string
	isCheapPrice bool
}

func (m MockTariff) Rates() ([]api.Rate, error) {
	var r []float64
	start := m.start

	switch m.feature {
	case "falling":
		r = []float64{5, 4, 3, 2, 1, 0, 0, 0}
	case "raising":
		r = []float64{1, 2, 3, 4, 5, 6, 7, 8}
	default:
		r = []float64{}
	}

	var res []api.Rate
	for _, v := range r {
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

func (m MockTariff) IsCheap() (bool, error) {
	return m.isCheapPrice, nil
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
		desc        string
		feature     string
		cheapactive bool
		end         time.Duration
		series      []se
	}{
		{"falling prices, 20min", "falling", false, 5 * time.Hour, []se{
			{1, 1*dt - 1, 20 * time.Minute, false},
			{2, 2*dt - 1, 20 * time.Minute, false},
			{3, 3*dt - 1, 20 * time.Minute, false},
			{4, 3*dt + 1, 20 * time.Minute, false},
			{5, 4*dt - 1, 20 * time.Minute, false},
			{6, 4*dt - 30*time.Minute, 20 * time.Minute, false}, // start as late as possible
			{7, 5*dt - 20*time.Minute, 20 * time.Minute, true},
			{8, 5*dt + 1, 5 * time.Minute, false}, // after desired charge timer,
		}},
		{"raising prices, 1h", "raising", false, 5 * time.Hour, []se{
			{1, 1*dt - 1, time.Hour, true},
			{2, 2*dt - 1, 5 * time.Minute, true}, // charging took longer than expected
			{3, 3*dt - 1, 0, false},
			{4, 5*dt + 1, 0, false}, // after desired charge timer
		}},
		{"after known prices, 1h", "falling", false, 5 * time.Hour, []se{
			{1, 20 * dt, time.Hour, false},
		}},
	}

	clck := clock.NewMock()
	start := clck.Now()
	for _, tc := range tc {
		clck.Set(start)

		p := &Planner{
			log:         util.NewLogger("foo"),
			clock:       clck,
			cheapactive: tc.cheapactive,
			tariff:      MockTariff{feature: tc.feature, start: start, isCheapPrice: false},
		}

		for _, se := range tc.series {
			clck.Set(start.Add(se.delay))

			if res, _ := p.isCheapSlotNow(se.cDuration, start.Add(tc.end)); se.res != res {
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
		desc        string
		feature     string
		cheapactive bool
		end         time.Duration
		isCheap     bool
		series      []se
	}{
		{"always cheap", "falling", false, 5 * time.Hour, true, []se{
			{1, 1*dt - 1, time.Minute, true},
			{2, 2*dt - 1, time.Minute, true},
			{3, 3*dt - 1, time.Minute, true},
			{4, 4*dt - 1, time.Minute, true},
			{5, 5*dt - 1, time.Minute, true},
		}},
		{"always expensive", "falling", true, 5 * time.Hour, false, []se{
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
		clck.Set(start)

		p := &Planner{
			log:         util.NewLogger("foo"),
			clock:       clck,
			cheapactive: tc.cheapactive,
			tariff:      MockTariff{feature: tc.feature, start: start, isCheapPrice: tc.isCheap},
		}

		for _, se := range tc.series {
			clck.Set(start.Add(se.delay))

			if res, _ := p.IsCheap(se.cDuration, start.Add(tc.end)); se.res != res {
				t.Errorf("%s case %v: expected %v, got %v", tc.desc, se.caseNr, se.res, res)
			}
		}
	}
}
