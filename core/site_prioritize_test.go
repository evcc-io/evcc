package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

const testEnableDelay = time.Minute

func newPVLoadpoint(prio int, mode api.ChargeMode, status api.ChargeStatus, enabled bool, timer time.Time) *Loadpoint {
	lp := &Loadpoint{
		log:        util.NewLogger("lp"),
		clock:      clock.NewMock(),
		minCurrent: minA,
		maxCurrent: maxA,
		phases:     1,
		mode:       mode,
		status:     status,
		enabled:    enabled,
		pvTimer:    timer,
		priority:   prio,
	}
	lp.Enable.Delay = testEnableDelay
	return lp
}

// pvTimerStarted returns a timer start that has been running for the given duration
func pvTimerStarted(running time.Duration) time.Time {
	return clock.NewMock().Now().Add(-running)
}

func TestPvChargeStarting(t *testing.T) {
	now := clock.NewMock().Now()
	settled := pvTimerStarted(testEnableDelay / 2)

	// enable timer running but car already full (soc at default 100% limit): not starting up
	enablePendingFull := newPVLoadpoint(0, api.ModePV, api.StatusB, false, settled)
	enablePendingFull.vehicleSoc = 100

	tc := []struct {
		name     string
		lp       *Loadpoint
		starting bool
	}{
		{"enable timer running", newPVLoadpoint(0, api.ModePV, api.StatusB, false, settled), true},
		{"enable timer just restarted", newPVLoadpoint(0, api.ModePV, api.StatusB, false, now), false},
		{"enabled not charging", newPVLoadpoint(0, api.ModePV, api.StatusB, true, time.Time{}), false},
		{"enabled and charging", newPVLoadpoint(0, api.ModePV, api.StatusC, true, time.Time{}), false},
		{"disabled idle", newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{}), false},
		{"disconnected", newPVLoadpoint(0, api.ModePV, api.StatusA, false, settled), false},
		{"not pv mode", newPVLoadpoint(0, api.ModeNow, api.StatusB, false, settled), false},
		{"enable pending but car full", enablePendingFull, false},
	}

	for _, tc := range tc {
		if got := tc.lp.PvChargeStarting(); got != tc.starting {
			t.Errorf("%s: want %v, got %v", tc.name, tc.starting, got)
		}
	}
}

func TestReservedPVPower(t *testing.T) {
	Voltage = 230

	// higher-priority loadpoint (prio 1) starting up
	high := newPVLoadpoint(1, api.ModePV, api.StatusB, false, pvTimerStarted(testEnableDelay/2))
	// lower-priority loadpoint (prio 0) in PV mode
	low := newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{})

	site := &Site{
		log:        util.NewLogger("site"),
		loadpoints: []*Loadpoint{high, low},
	}

	// low reserves the power high needs to start, not its max power (#32778)
	if got, want := site.reservedPVPower(low), high.EffectiveMinPower(); got != want {
		t.Errorf("low: want %.0f, got %.0f", want, got)
	}
	if high.EffectiveMinPower() == high.EffectiveMaxPower() {
		t.Fatal("test requires min and max power to differ")
	}

	// a timer restarting on every surplus dip must not reserve at all (#32778)
	high.pvTimer = clock.NewMock().Now()
	if got := site.reservedPVPower(low); got != 0 {
		t.Errorf("low while high timer restarts: want 0, got %.0f", got)
	}
	high.pvTimer = pvTimerStarted(testEnableDelay / 2)

	// high (top priority) reserves nothing
	if got := site.reservedPVPower(high); got != 0 {
		t.Errorf("high: want 0, got %.0f", got)
	}

	// once high is charging it no longer reserves surplus from low
	high.status = api.StatusC
	high.enabled = true
	high.pvTimer = time.Time{}
	if got := site.reservedPVPower(low); got != 0 {
		t.Errorf("low after high charging: want 0, got %.0f", got)
	}

	// high stays enabled and connected but no longer draws (car full): no reservation (#31684)
	high.status = api.StatusB
	if got := site.reservedPVPower(low); got != 0 {
		t.Errorf("low after high stopped drawing: want 0, got %.0f", got)
	}
}

var _ loadpoint.API = (*Loadpoint)(nil)
