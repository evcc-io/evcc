package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

func newPVLoadpoint(prio int, mode api.ChargeMode, status api.ChargeStatus, enabled bool, timer time.Time) *Loadpoint {
	return &Loadpoint{
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
}

func TestPvChargeStarting(t *testing.T) {
	now := clock.NewMock().Now()

	// enable timer running but car already full (soc at default 100% limit): not starting up
	enablePendingFull := newPVLoadpoint(0, api.ModePV, api.StatusB, false, now)
	enablePendingFull.vehicleSoc = 100

	tc := []struct {
		name     string
		lp       *Loadpoint
		starting bool
	}{
		{"enable timer running", newPVLoadpoint(0, api.ModePV, api.StatusB, false, now), true},
		{"enabled not charging", newPVLoadpoint(0, api.ModePV, api.StatusB, true, time.Time{}), false},
		{"enabled and charging", newPVLoadpoint(0, api.ModePV, api.StatusC, true, time.Time{}), false},
		{"disabled idle", newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{}), false},
		{"disconnected", newPVLoadpoint(0, api.ModePV, api.StatusA, false, now), false},
		{"not pv mode", newPVLoadpoint(0, api.ModeNow, api.StatusB, false, now), false},
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
	high := newPVLoadpoint(1, api.ModePV, api.StatusB, false, clock.NewMock().Now())
	// lower-priority loadpoint (prio 0) in PV mode
	low := newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{})

	site := &Site{
		log:        util.NewLogger("site"),
		loadpoints: []*Loadpoint{high, low},
	}

	// low reserves high's anticipated max power while high is starting up
	if got, want := site.reservedPVPower(low), high.EffectiveMaxPower(); got != want {
		t.Errorf("low: want %.0f, got %.0f", want, got)
	}

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
