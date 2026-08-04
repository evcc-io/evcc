package planner

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
)

// SharedPlanRequest is one loadpoint's input for circuit-aware allocation.
type SharedPlanRequest struct {
	// Rank is the loadpoint's circuit arbitration rank (api.CircuitLoad.GetRank).
	// Allocating in the same order the circuit clamp enforces at runtime keeps the
	// plan's assumed share and the share it actually gets in agreement.
	Rank             int
	MaxPower         float64
	MinPower         float64 // effectiveMinPower, semi-continuous floor
	RequiredDuration time.Duration
	TargetTime       time.Time
}

// AllocateShared plans loadpoints sharing one circuit against a per-slot power
// budget in descending rank, each scheduled only where its MinPower fits and
// reserving its actual draw. Returns one plan per request, in input order.
// Clamped to [now, target] per request; single circuit + static budget.
func AllocateShared(now time.Time, budget float64, rates api.Rates, reqs []SharedPlanRequest) []api.Rates {
	ledger := NewCapacityLedger(budget, tariff.SlotDuration)

	// plan the highest rank first, keeping input order for the result
	order := make([]int, len(reqs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return reqs[b].Rank - reqs[a].Rank
	})

	plans := make([]api.Rates, len(reqs))
	for _, i := range order {
		r := reqs[i]

		// restrict to this request's [now, target] window, then cheapest-first
		sorted := clampRates(rates, now, r.TargetTime)
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
