package planner

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// CapacityLedger tracks residual charging power available per time slot on a
// shared circuit, so loadpoints can be planned in priority order without
// overcommitting the circuit. Slots are keyed by their truncated start time.
type CapacityLedger struct {
	slot   time.Duration
	budget float64               // total circuit power (W) per slot
	used   map[time.Time]float64 // reserved power (W) per slot start
}

// NewCapacityLedger creates a ledger with the given per-slot power budget.
func NewCapacityLedger(budget float64, slot time.Duration) *CapacityLedger {
	return &CapacityLedger{slot: slot, budget: budget, used: make(map[time.Time]float64)}
}

// Available returns the residual power in the slot containing t.
func (l *CapacityLedger) Available(t time.Time) float64 {
	return max(0, l.budget-l.used[t.Truncate(l.slot)])
}

// Reserve subtracts power across every slot the plan occupies.
func (l *CapacityLedger) Reserve(plan api.Rates, power float64) {
	for _, r := range plan {
		for t := r.Start.Truncate(l.slot); t.Before(r.End); t = t.Add(l.slot) {
			l.used[t] += power
		}
	}
}

// CanHost reports whether the slot containing t can still grant minPower.
// Semi-continuous: a charger runs at >= minPower or off (evcc-io/optimizer#91).
func (l *CapacityLedger) CanHost(t time.Time, minPower float64) bool {
	return l.Available(t) >= minPower
}
