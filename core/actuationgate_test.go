package core

import (
	"errors"
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// TestActuationGate covers the gate in isolation: first acquire grants, a second
// within the lock is denied with the remaining duration, and after the lock
// expires it grants again. A non-positive lock always grants.
func TestActuationGate(t *testing.T) {
	clk := clock.NewMock()
	g := newActuationGate(clk, time.Minute)

	if granted, _ := g.tryAcquire(); !granted {
		t.Fatal("first acquire must be granted")
	}

	clk.Add(20 * time.Second)
	if granted, retryAfter := g.tryAcquire(); granted || retryAfter != 40*time.Second {
		t.Fatalf("within lock: granted=%v retryAfter=%v, want false / 40s", granted, retryAfter)
	}

	clk.Add(40 * time.Second) // now exactly at the lock boundary
	if granted, _ := g.tryAcquire(); !granted {
		t.Fatal("acquire after lock expiry must be granted")
	}

	// non-positive lock disables gating
	g0 := newActuationGate(clk, 0)
	if granted, _ := g0.tryAcquire(); !granted {
		t.Fatal("zero lock must always grant")
	}
}

func newGateLoadpoint(t *testing.T, charger api.Charger, clk clock.Clock, gate *actuationGate) *Loadpoint {
	t.Helper()
	return &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clk,
		bus:         evbus.New(),
		charger:     charger,
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		gate:        gate,
	}
}

// TestSetLimitGateDefersIncrease verifies that a budget-increasing actuation is
// deferred while the settle lock is active and proceeds once it expires.
func TestSetLimitGateDefersIncrease(t *testing.T) {
	clk := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := newGateLoadpoint(t, charger, clk, newActuationGate(clk, time.Minute))

	// initial enable from disabled: grants (gate fresh)
	charger.EXPECT().MaxCurrent(int64(minA)).Return(nil)
	charger.EXPECT().Enable(true).Return(nil)
	if err := lp.setLimit(minA); err != nil {
		t.Fatalf("initial setLimit: %v", err)
	}

	// raise current immediately: within lock -> deferred, no charger calls
	if err := lp.setLimit(maxA); err != nil {
		t.Fatalf("deferred setLimit must be a silent no-op, got %v", err)
	}
	if lp.offeredCurrent != minA {
		t.Fatalf("offeredCurrent changed despite deferral: %v", lp.offeredCurrent)
	}
	if !lp.pendingControl.Load() {
		t.Fatal("deferral must mark the loadpoint pending")
	}

	// after the lock expires the raise proceeds
	clk.Add(time.Minute)
	charger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)
	if err := lp.setLimit(maxA); err != nil {
		t.Fatalf("setLimit after lock: %v", err)
	}
	if lp.offeredCurrent != maxA {
		t.Fatalf("offeredCurrent not raised: %v", lp.offeredCurrent)
	}
}

// TestSetLimitGateBypassesDecrease verifies that reductions are never blocked by
// the settle lock, even while it is active.
func TestSetLimitGateBypassesDecrease(t *testing.T) {
	clk := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	gate := newActuationGate(clk, time.Minute)
	gate.tryAcquire() // stamp: gate is now locked

	lp := newGateLoadpoint(t, charger, clk, gate)
	lp.enabled = true
	lp.offeredCurrent = maxA

	// decrease while the lock is active must go through immediately
	charger.EXPECT().MaxCurrent(int64(minA)).Return(nil)
	if err := lp.setLimit(minA); err != nil {
		t.Fatalf("decrease setLimit: %v", err)
	}
	if lp.offeredCurrent != minA {
		t.Fatalf("offeredCurrent not reduced: %v", lp.offeredCurrent)
	}
}

// TestScalePhasesGateBothDirections verifies that phase switches acquire the gate
// in both directions (a switch causes a charging pause that must hold off others).
func TestScalePhasesGateBothDirections(t *testing.T) {
	clk := clock.NewMock()
	ctrl := gomock.NewController(t)

	phaseSwitcher := api.NewMockPhaseSwitcher(ctrl)
	charger := struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{
		api.NewMockCharger(ctrl), phaseSwitcher,
	}

	gate := newActuationGate(clk, time.Minute)
	gate.tryAcquire() // stamp: gate is now locked

	lp := newGateLoadpoint(t, charger, clk, gate)
	lp.phases = 1

	// scale-up within lock -> deferred, no switch
	if err := lp.scalePhases(3); !errors.Is(err, errActuationDeferred) {
		t.Fatalf("scale-up within lock: want errActuationDeferred, got %v", err)
	}

	// after lock expiry the up-switch proceeds (and re-stamps the gate)
	clk.Add(time.Minute)
	phaseSwitcher.EXPECT().Phases1p3p(3).Return(nil)
	if err := lp.scalePhases(3); err != nil {
		t.Fatalf("scale-up after lock: %v", err)
	}

	// down-switch immediately after is gated too -> deferred, no Phases1p3p(1)
	if err := lp.scalePhases(1); !errors.Is(err, errActuationDeferred) {
		t.Fatalf("scale-down within lock: want errActuationDeferred, got %v", err)
	}
}

// loopLoadpoints scheduler tests live in scheduler_test.go (the scheduler
// outlives the actuation gate, which is removed in a later phase).
