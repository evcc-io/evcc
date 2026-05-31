package core

import (
	"testing"

	"github.com/evcc-io/evcc/util"
)

// TestLoopLoadpointsPrefersPending verifies the scheduler hands out a pending
// loadpoint before advancing the round-robin cursor.
func TestLoopLoadpointsPrefersPending(t *testing.T) {
	lp0 := &Loadpoint{log: util.NewLogger("lp0")}
	lp1 := &Loadpoint{log: util.NewLogger("lp1")}

	site := &Site{
		log:        util.NewLogger("site"),
		loadpoints: []*Loadpoint{lp0, lp1},
	}

	lp1.pendingControl.Store(true)

	next := make(chan updater)
	go site.loopLoadpoints(next)

	if got := <-next; got != lp1 {
		t.Fatalf("first dispatch: want pending lp1, got %v", got)
	}
	if lp1.pendingControl.Load() {
		t.Fatal("pending flag must be cleared on dispatch")
	}
	if got := <-next; got != lp0 {
		t.Fatalf("second dispatch: want round-robin lp0, got %v", got)
	}
}

// TestLoopLoadpointsNoStarvation verifies a perpetually-pending loadpoint cannot
// starve the round-robin sweep that keeps idle loadpoints regulated.
func TestLoopLoadpointsNoStarvation(t *testing.T) {
	lp0 := &Loadpoint{log: util.NewLogger("lp0")}
	lp1 := &Loadpoint{log: util.NewLogger("lp1")}

	site := &Site{
		log:        util.NewLogger("site"),
		loadpoints: []*Loadpoint{lp0, lp1},
	}

	lp0.pendingControl.Store(true)

	next := make(chan updater)
	go site.loopLoadpoints(next)

	var seenLp1 bool
	for i := 0; i < 6; i++ {
		got := <-next
		if got == lp1 {
			seenLp1 = true
		}
		lp0.pendingControl.Store(true) // simulate a perpetual pending source
	}

	if !seenLp1 {
		t.Fatal("round-robin starved: lp1 never dispatched while lp0 stayed pending")
	}
}
