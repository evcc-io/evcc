package core

import (
	"testing"

	"github.com/evcc-io/evcc/util/log"
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

	log := log.NewLogger("foo")

	for _, tc := range tc {
		res := sitePower(log, tc.maxGrid, tc.grid, tc.battery, 0)
		if res != tc.site {
			t.Errorf("sitePower wanted %.f, got %.f", tc.site, res)
		}
	}
}
