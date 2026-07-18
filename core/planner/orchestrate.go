package planner

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
)

// SharedPlanRequest is one loadpoint's input for circuit-aware allocation.
type SharedPlanRequest struct {
	Forced           bool // minSoc/ModeNow: reserved before cost optimisation
	Priority         int
	MaxPower         float64
	MinPower         float64 // effectiveMinPower, semi-continuous floor
	RequiredDuration time.Duration
	TargetTime       time.Time
}

// AllocateShared plans loadpoints sharing one circuit against a per-slot power
// budget: forced first, then descending priority, each scheduled only where its
// MinPower fits and reserving its actual draw. Returns one plan per request, in
// input order. Rates must be pre-clamped; single circuit + static budget only.
func AllocateShared(budget float64, rates api.Rates, reqs []SharedPlanRequest) []api.Rates {
	ledger := NewCapacityLedger(budget, tariff.SlotDuration)

	// plan forced first, then highest priority, keeping input order for the result
	order := make([]int, len(reqs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		x, y := reqs[a], reqs[b]
		if x.Forced != y.Forced {
			if x.Forced {
				return -1
			}
			return 1
		}
		return y.Priority - x.Priority
	})

	plans := make([]api.Rates, len(reqs))
	for _, i := range order {
		r := reqs[i]

		sorted := slices.Clone(rates)
		slices.SortStableFunc(sorted, sortByCost)

		avail := func(t time.Time) float64 { return ledger.Available(t) }
		p := planCapped(sorted, r.RequiredDuration, r.TargetTime, avail, r.MaxPower, r.MinPower)
		p.Sort()

		// reserve the actual per-slot draw (min of MaxPower and residual), matching
		// what planCapped assumed, so the next loadpoint sees the real remainder
		for _, s := range p {
			ledger.Reserve(api.Rates{s}, min(r.MaxPower, ledger.Available(s.Start)))
		}

		plans[i] = p
	}

	return plans
}
