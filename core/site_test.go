package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/util"
	"github.com/golang/mock/gomock"
)

func TestSitePower(t *testing.T) {
	tc := []struct {
		grid, battery, site float64
	}{
		{0, 0, 0},    // silent night
		{0, 1, 1},    // battery discharging
		{0, -1, -1},  // battery charging -> negative result cannot occur in reality
		{1, 0, 1},    // grid import
		{1, 1, 2},    // grid import + battery discharging
		{-1, 0, -1},  // grid export
		{-1, -1, -2}, // grid export + battery charging
	}

	for _, tc := range tc {
		res := sitePower(tc.grid, tc.battery, 0)
		if res != tc.site {
			t.Errorf("sitePower wanted %.f, got %.f", tc.site, res)
		}
	}
}

func TestSiteUpdateMultipleLoadpoints(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sitePower := -1000.0
	lps := []float64{300.0, 400.0, 500.0}

	gm := mock.NewMockMeter(ctrl)
	gm.EXPECT().CurrentPower().Return(sitePower, nil)

	site := &Site{
		log:       util.NewLogger("foo"),
		Mode:      api.ModePV,
		gridMeter: gm,
	}

	availablePower := sitePower

	for _, usedPower := range lps {
		lp := mock.NewMockLoadPointer(ctrl)
		site.loadpoints = append(site.loadpoints, lp)

		lp.EXPECT().Update(api.ModePV, availablePower).Return(usedPower)
		availablePower += usedPower
	}

	_ = site.update()
}
