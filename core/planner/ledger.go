package planner

import (
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

// Owner identifies a loadpoint planning against shared circuit capacity
type Owner struct {
	Id       int         // loadpoint sequence, breaks the last tie
	Priority int         // higher priority plans first
	Target   time.Time   // earlier target plans first at equal priority
	Circuit  api.Circuit // circuit the loadpoint is attached to, nil = unlimited
	MaxPower float64     // power the loadpoint plans with
	MinPower float64     // below this share a slot is useless
}

// outranks reports whether o plans before other, i.e. other must plan around o
func (o Owner) outranks(other Owner) bool {
	if o.Priority != other.Priority {
		return o.Priority > other.Priority
	}
	if !o.Target.Equal(other.Target) {
		return o.Target.Before(other.Target)
	}
	return o.Id < other.Id
}

type reservation struct {
	owner Owner
	plan  api.Rates
}

// Ledger tracks the plans of all loadpoints so circuit capacity is shared between them.
// A loadpoint plans around the reservations of every loadpoint that outranks it and
// ignores the rest, so the ranking converges within one cycle regardless of update order.
type Ledger struct {
	mu           sync.Mutex
	reservations map[int]reservation
}

// NewLedger creates a ledger
func NewLedger() *Ledger {
	return &Ledger{reservations: make(map[int]reservation)}
}

// Reserve records the plan for the owner, an empty plan releases the reservation
func (l *Ledger) Reserve(owner Owner, plan api.Rates) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(plan) == 0 {
		delete(l.reservations, owner.Id)
		return
	}

	l.reservations[owner.Id] = reservation{owner, plan}
}

// Available returns the power left for the owner during the slot on every circuit
// from its own up to the root, or +Inf without any limit
func (l *Ledger) Available(owner Owner, slot api.Rate) float64 {
	res := math.Inf(1)

	l.mu.Lock()
	defer l.mu.Unlock()

	for c := owner.Circuit; c != nil; c = c.GetParent() {
		maxPower := c.GetMaxPower()
		if maxPower <= 0 {
			continue
		}

		var used float64
		for _, r := range l.reservations {
			if r.owner.Id == owner.Id || !r.owner.outranks(owner) || !belongsTo(r.owner.Circuit, c) {
				continue
			}
			used += overlapPower(r.plan, slot)
		}

		res = min(res, max(0, maxPower-used))
	}

	return res
}

// belongsTo reports whether circuit is c or a descendant of c
func belongsTo(circuit, c api.Circuit) bool {
	for ; circuit != nil; circuit = circuit.GetParent() {
		if circuit == c {
			return true
		}
	}
	return false
}

// overlapPower returns the highest power the plan draws while the slot is active
func overlapPower(plan api.Rates, slot api.Rate) float64 {
	var res float64
	for _, r := range plan {
		if r.Start.Before(slot.End) && r.End.After(slot.Start) {
			res = max(res, r.Power)
		}
	}
	return res
}
