package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	//"github.com/golang/mock/gomock"
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
		r = []float64{5, 4, 3, 2, 1}
	case "raising":
		r = []float64{1, 2, 3, 4, 5}
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

func (m MockTariff) isCheap() (bool, error) {
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
			{6, 5*dt - 20*time.Minute, 20 * time.Minute, true},
		}},
		{"raising prices, 1h", "raising", false, 5 * time.Hour, []se{
			{1, 1*dt - 1, time.Hour, true},
			{2, 2*dt - 1, 5 * time.Minute, true}, // charging took longer than expected
			{3, 3*dt - 1, 0, false},
		}},
	}

	clck := clock.NewMock()
	start := clck.Now()
	for _, tc := range tc {
		//t.Logf("%+v", tc)
		clck.Set(start)
		fmt.Printf("set start to: %s\n", start)

		p := &Planner{
			log:         util.NewLogger("foo"),
			clock:       clck,
			cheapactive: tc.cheapactive,
			tariff:      MockTariff{feature: tc.feature, start: start, isCheapPrice: false},
		}

		for _, se := range tc.series {
			fmt.Printf("clk set to: %s\n", start.Add(se.delay))
			clck.Set(start.Add(se.delay))

			if res, _ := p.isCheapSlotNow(se.cDuration, start.Add(tc.end)); se.res != res {
				t.Errorf("%s case %v: expected %v, got %v", tc.desc, se.caseNr, se.res, res)
			}
		}
	}
}
