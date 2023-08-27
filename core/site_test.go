package core

import (
	"testing"

	"github.com/evcc-io/evcc/util"
)

func TestSitePower(t *testing.T) {
	tc := []struct {
		maxGrid, grid, battery, site float64
	}{
		// {0, 0, 0, 0},    // silent night
		// {0, 0, 1, 1},    // battery discharging
		// {0, 0, -1, -1},  // battery charging -> negative result cannot occur in reality
		// {0, 1, 0, 1},    // grid import
		// {0, 1, 1, 2},    // grid import + battery discharging
		// {0, -1, 0, -1},  // grid export
		// {0, -1, -1, -2}, // grid export + battery charging
		{0, 1, -1, 0},   // grid import + battery charging -> should not happen
		{0.5, 1, -1, 1}, // grid import + DC battery charging
	}

	log := util.NewLogger("foo")

	for _, tc := range tc {
		res := sitePower(log, tc.maxGrid, tc.grid, tc.battery, 0)
		if res != tc.site {
			t.Errorf("sitePower wanted %.f, got %.f", tc.site, res)
		}
	}
}

func TestGreenShare(t *testing.T) {
	tc := []struct {
		title                                                 string
		grid, pv, battery, home, lp                           float64
		greenShareTotal, greenShareHome, greenShareLoadpoints float64
	}{
		{"half grid, half pv",
			2500, 2500, 0, 1000, 4000,
			0.5, 1, 1500 / 4000},
		{"half grid, half pv, no lp",
			2500, 2500, 0, 5000, 0,
			0.5, 0.5, 0},
		{"full pv",
			0, 5000, 0, 1000, 4000,
			1, 1, 1},
		{"full grid",
			5000, 0, 0, 1000, 4000,
			0, 0, 0},
		{"half grid, half battery",
			2500, 0, 2500, 1000, 4000,
			0.5, 1, 1500 / 4000},
		{"full pv, pv export",
			-5000, 10000, 0, 1000, 4000,
			1, 1, 1},
		{"full pv, pv export, no lp",
			-5000, 10000, 0, 5000, 0,
			1, 1, 1},
		{"full pv, pv export, battery charge",
			-2500, 10000, -2500, 1000, 4000,
			1, 1, 1},
		{"double charge speed, full grid",
			10000, 0, 0, 1000, 4000,
			0, 0, 0},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		s := &Site{
			gridPower:    tc.grid,
			pvPower:      tc.pv,
			batteryPower: tc.battery,
		}

		greenShareTotal := s.greenShare(0, tc.home+tc.lp)
		if greenShareTotal != tc.greenShareTotal {
			t.Errorf("greenShareTotal wanted %.2f, got %.2f", tc.greenShareTotal, greenShareTotal)
		}
		greenShareHome := s.greenShare(0, tc.home)
		if greenShareHome != tc.greenShareHome {
			t.Errorf("greenShareHome wanted %.2f, got %.2f", tc.greenShareHome, greenShareHome)
		}
		greenShareLoadpoints := s.greenShare(tc.home, tc.home+tc.lp)
		if greenShareLoadpoints != tc.greenShareLoadpoints {
			t.Errorf("greenShareLoadpoints wanted %.2f, got %.2f", tc.greenShareLoadpoints, greenShareLoadpoints)
		}
	}
}
