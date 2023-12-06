package core

import (
	"testing"

	"github.com/evcc-io/evcc/util"
)

func TestSitePower(t *testing.T) {
	tc := []struct {
		maxGrid, grid, battery, site float64
	}{
		{0, 0, 0, 0},    // silent night
		{0, 0, 1, 1},    // battery discharging
		{0, 0, -1, -1},  // battery charging -> negative result cannot occur in reality
		{0, 1, 0, 1},    // grid import
		{0, 1, 1, 2},    // grid import + battery discharging
		{0, -1, 0, -1},  // grid export
		{0, -1, -1, -2}, // grid export + battery charging
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
		{
			"half grid, half pv, green home",
			1000, 1000, 0, 1000, 1000,
			0.5, 1, 0,
		},
		{
			"half grid, half pv, no home",
			1000, 1000, 0, 0, 2000,
			0.5, 1, 0.5,
		},
		{
			"half grid, half pv, no lp",
			2500, 2500, 0, 5000, 0,
			0.5, 0.5, 0,
		},
		{
			"full pv",
			0, 5000, 0, 1000, 4000,
			1, 1, 1,
		},
		{
			"full grid",
			5000, 0, 0, 1000, 4000,
			0, 0, 0,
		},
		{
			"half grid, half battery, green home",
			1000, 0, 1000, 1000, 1000,
			0.5, 1, 0,
		},
		{
			"half grid, half battery, no home",
			1000, 0, 1000, 0, 2000,
			0.5, 1, 0.5,
		},
		{
			"half grid, half battery, no lp",
			1000, 0, 1000, 2000, 0,
			0.5, 0.5, 0,
		},
		{
			"full pv, pv export",
			-5000, 10000, 0, 1000, 4000,
			1, 1, 1,
		},
		{
			"full pv, pv export, no lp",
			-5000, 10000, 0, 5000, 0,
			1, 1, 1,
		},
		{
			"full pv, pv export, battery charge",
			-2500, 10000, -2500, 1000, 4000,
			1, 1, 1,
		},
		{
			"full grid, battery charge",
			3000, 0, -1000, 1000, 1000,
			0, 0, 0,
		},
		{
			"full grid, battery charge, no lp",
			2000, 0, -1000, 1000, 0,
			0, 0, 0,
		},
		{
			"half grid, half pv, battery charge, no lp",
			1000, 1000, -1000, 1000, 0,
			1, 1, 0,
		},
		{
			"half grid, half pv, battery charge, home, lp",
			1000, 1000, -1000, 500, 500,
			1, 1, 1,
		},
	}

	for _, tc := range tc {
		t.Logf(tc.title)

		s := &Site{
			gridPower:    tc.grid,
			pvPower:      tc.pv,
			batteryPower: tc.battery,
		}

		greenShareTotal := s.greenShare(0, tc.home+tc.lp)
		if greenShareTotal != tc.greenShareTotal {
			t.Errorf("greenShareTotal wanted %.3f, got %.3f", tc.greenShareTotal, greenShareTotal)
		}
		greenShareHome := s.greenShare(0, tc.home)
		if greenShareHome != tc.greenShareHome {
			t.Errorf("greenShareHome wanted %.3f, got %.3f", tc.greenShareHome, greenShareHome)
		}
		greenShareLoadpoints := s.greenShare(tc.home, tc.home+tc.lp)
		if greenShareLoadpoints != tc.greenShareLoadpoints {
			t.Errorf("greenShareLoadpoints wanted %.3f, got %.3f", tc.greenShareLoadpoints, greenShareLoadpoints)
		}
	}
}
