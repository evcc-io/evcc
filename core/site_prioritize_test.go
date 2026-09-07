package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
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
	enablePendingFull := newPVLoadpoint(0, api.ModeSmart, api.StatusB, false, now)
	enablePendingFull.vehicleSoc = 100

	tc := []struct {
		name     string
		lp       *Loadpoint
		starting bool
	}{
		{"enable timer running", newPVLoadpoint(0, api.ModeSmart, api.StatusB, false, now), true},
		{"enabled not charging", newPVLoadpoint(0, api.ModeSmart, api.StatusB, true, time.Time{}), false},
		{"enabled and charging", newPVLoadpoint(0, api.ModeSmart, api.StatusC, true, time.Time{}), false},
		{"disabled idle", newPVLoadpoint(0, api.ModeSmart, api.StatusB, false, time.Time{}), false},
		{"disconnected", newPVLoadpoint(0, api.ModeSmart, api.StatusA, false, now), false},
		{"not pv mode", newPVLoadpoint(0, api.ModeNow, api.StatusB, false, now), false},
		{"enable pending but car full", enablePendingFull, false},
		// elapsed means a delay was skipped, e.g. by a feed-in pause, not an enable pending
		{"timer elapsed", newPVLoadpoint(0, api.ModeSmart, api.StatusB, false, elapsed), false},
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
	high := newPVLoadpoint(1, api.ModeSmart, api.StatusB, false, clock.NewMock().Now())
	// lower-priority loadpoint (prio 0) in PV mode
	low := newPVLoadpoint(0, api.ModeSmart, api.StatusB, false, time.Time{})

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

// feedInCharger is a stateful charger, unlike the mocks used elsewhere, since the
// feed-in pause is asserted across multiple update cycles
type feedInCharger struct {
	status  api.ChargeStatus
	enabled bool
}

func (c *feedInCharger) Status() (api.ChargeStatus, error) { return c.status, nil }
func (c *feedInCharger) Enabled() (bool, error)            { return c.enabled, nil }
func (c *feedInCharger) MaxCurrent(int64) error            { return nil }

func (c *feedInCharger) Enable(v bool) error {
	c.enabled = v
	// a disabled charger stops drawing, i.e. reports connected instead of charging
	if !v && c.status == api.StatusC {
		c.status = api.StatusB
	}
	return nil
}

// TestReservedPVPowerSmartFeedInPause asserts that a loadpoint paused by its smart feed-in
// priority limit does not reserve surplus: it is meant to export instead of charge
func TestReservedPVPowerSmartFeedInPause(t *testing.T) {
	// the feed-in rate rises above the limit while the loadpoint is idle or charging
	for _, tc := range []struct {
		name    string
		status  api.ChargeStatus
		enabled bool
	}{
		{"idle", api.StatusB, false},
		{"charging", api.StatusC, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clck := clock.NewMock()
			limit := 0.05

			car := newPVLoadpoint(1, api.ModeSmart, tc.status, tc.enabled, time.Time{})
			car.bus = evbus.New()
			car.clock = clck
			car.charger = &feedInCharger{status: tc.status, enabled: tc.enabled}
			car.chargeMeter = &Null{} // silence nil panics
			car.chargeRater = &Null{} // silence nil panics
			car.chargeTimer = &Null{} // silence nil panics
			car.wakeUpTimer = NewTimer()
			car.smartFeedInPriorityLimit = &limit
			car.vehicleSoc = 20
			car.limitSoc = 80
			attachListeners(t, car)

			low := newPVLoadpoint(0, api.ModeSmart, api.StatusB, false, time.Time{})

			site := &Site{
				log:        util.NewLogger("site"),
				loadpoints: []*Loadpoint{car, low},
			}

			// checkSmartLimit looks up rates by wall clock time, not lp.clock
			now := time.Now()
			feedin := api.Rates{{Start: now.Add(-time.Hour), End: now.Add(time.Hour), Value: limit + 0.01}}

			// the pause holds across cycles, so must the absence of a reservation
			for i := range 2 {
				car.Update(-3500, 0, nil, feedin, false, false, 0, nil, nil, nil)

				if car.enabled {
					t.Fatalf("cycle %d: car must be paused by the feed-in limit", i)
				}

				if got := site.reservedPVPower(low); got != 0 {
					t.Errorf("cycle %d: paused car reserves %.0fW, want 0W (pvTimer %v)", i, got, car.pvTimer)
				}

				clck.Add(time.Minute)
			}
		})
	}
}

var _ loadpoint.API = (*Loadpoint)(nil)
