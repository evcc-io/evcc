package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// surplusClaim is a power change that has been actuated but is not yet reflected
// by the meters.
type surplusClaim struct {
	power float64
	at    time.Time
}

// surplusLedger tracks power claimed by loadpoint actuations but not yet
// reflected by the grid meters. It lets several loadpoints actuate in quick
// succession without overshooting: each claim discounts the surplus seen by the
// next loadpoint until the meters catch up. A claim is "in flight" only while it
// is younger than settle (after that the meters include it), so the ledger is
// self-reconciling and needs no explicit reset.
type surplusLedger struct {
	mu     sync.Mutex
	clock  clock.Clock
	settle time.Duration
	claims []surplusClaim
}

// newSurplusLedger returns a ledger whose claims expire after settle.
func newSurplusLedger(clk clock.Clock, settle time.Duration) *surplusLedger {
	return &surplusLedger{clock: clk, settle: settle}
}

// claim records a power change (positive for an increase, negative for a
// decrease). A zero delta or a non-positive settle is ignored.
func (l *surplusLedger) claim(deltaPower float64) {
	if deltaPower == 0 || l.settle <= 0 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.prune()
	l.claims = append(l.claims, surplusClaim{power: deltaPower, at: l.clock.Now()})
}

// inflight returns the sum of claims the meters have not yet caught up on.
func (l *surplusLedger) inflight() float64 {
	if l.settle <= 0 {
		return 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.prune()

	var sum float64
	for _, c := range l.claims {
		sum += c.power
	}
	return sum
}

// prune drops expired claims. The caller must hold mu.
func (l *surplusLedger) prune() {
	cutoff := l.clock.Now().Add(-l.settle)

	keep := l.claims[:0]
	for _, c := range l.claims {
		if c.at.After(cutoff) {
			keep = append(keep, c)
		}
	}
	l.claims = keep
}
