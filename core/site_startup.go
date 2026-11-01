package core

import (
	"slices"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/samber/lo"
)

// startupOrder sorts loadpoints for the initial update: charging first,
// then those with a connected vehicle, then the remaining ones
func startupOrder(lps []*Loadpoint) []*Loadpoint {
	type ranked struct {
		lp   *Loadpoint
		rank int
	}

	res := make([]ranked, len(lps))

	// read the charger states concurrently, a slow charger must not delay the others
	var wg sync.WaitGroup
	for i, lp := range lps {
		wg.Go(func() { res[i] = ranked{lp, startupRank(lp)} })
	}
	wg.Wait()

	slices.SortStableFunc(res, func(a, b ranked) int { return a.rank - b.rank })

	return lo.Map(res, func(r ranked, _ int) *Loadpoint { return r.lp })
}

// startupRank ranks the loadpoint for the initial update after startup:
// 0 charging, 1 vehicle connected, 2 otherwise. The charge power is measured by the site update,
// the charger status is only read for ordering and not applied.
func startupRank(lp *Loadpoint) int {
	if lp.GetChargePower() > standbyPower {
		return 0
	}

	if status, err := lp.charger.Status(); err == nil && (status == api.StatusB || status == api.StatusC) {
		return 1
	}

	return 2
}
