package core

import (
	"errors"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// errActuationDeferred signals that an actuation was postponed by the settle
// lock. It is not a failure: the loadpoint schedules a retry and re-decides on
// the next pass. Callers treat it as a silent no-op.
var errActuationDeferred = errors.New("actuation deferred by settle lock")

// actuationGate enforces a minimum spacing ("settle lock") between
// budget-increasing actuations across the whole site. It decouples how often
// loadpoints are evaluated (fast, event-driven) from how often setpoints are
// actually changed (spaced by the lock), preventing several loadpoints from
// grabbing surplus before the meters reflect the previous change.
type actuationGate struct {
	mu    sync.Mutex
	clock clock.Clock
	lock  time.Duration
	last  time.Time
}

// newActuationGate returns a gate spacing actuations by lock, driven by clk.
func newActuationGate(clk clock.Clock, lock time.Duration) *actuationGate {
	return &actuationGate{clock: clk, lock: lock}
}

// tryAcquire reports whether an actuation may proceed now. On success it stamps
// the gate so subsequent actuations are held off for the lock duration; on
// denial it returns the remaining lock so the caller can schedule a retry.
func (g *actuationGate) tryAcquire() (bool, time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.lock <= 0 {
		return true, 0
	}

	if !g.last.IsZero() {
		if elapsed := g.clock.Now().Sub(g.last); elapsed < g.lock {
			return false, g.lock - elapsed
		}
	}

	g.last = g.clock.Now()
	return true, 0
}
