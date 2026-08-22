package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

const (
	testEnableDelay = time.Minute

	// feed-in priority pauses charging while the feed-in rate is at or above the limit
	feedInRate  = 0.075             // currency/kWh
	feedInLimit = feedInRate - 0.01 // configured limit, i.e. currently exceeded

	testSurplus = -3500.0 // 3.5kW PV surplus
)

// testNow is the reference time for all clocks and rates in these tests. It has to track
// the wall clock: checkSmartLimit looks up tariff rates by time.Now() instead of lp.clock
// (see loadpoint_smartcost.go). Truncating keeps derived times stable per run.
var testNow = time.Now().Truncate(time.Hour)

// newTestClock returns a mock clock at testNow
func newTestClock() *clock.Mock {
	clck := clock.NewMock()
	clck.Set(testNow)
	return clck
}

// newTestLoadpoint returns a single phase PV mode loadpoint on the given clock
func newTestLoadpoint(title string, clck *clock.Mock) *Loadpoint {
	lp := NewLoadpoint(util.NewLogger(title), nil)
	lp.clock = clck
	lp.mode = api.ModePV
	lp.minCurrent = minA
	lp.maxCurrent = maxA
	lp.phases = 1
	lp.Enable.Delay = testEnableDelay
	lp.Disable.Delay = 3 * testEnableDelay

	return lp
}

func newPVLoadpoint(prio int, mode api.ChargeMode, status api.ChargeStatus, enabled bool, timer time.Time) *Loadpoint {
	lp := newTestLoadpoint("lp", newTestClock())
	lp.mode = mode
	lp.status = status
	lp.enabled = enabled
	lp.pvTimer = timer
	lp.priority = prio

	return lp
}

// feedInCharger is a stateful charger for the tests that need the full Update loop
type feedInCharger struct {
	status  api.ChargeStatus
	enabled bool
	current int64
}

func (c *feedInCharger) Status() (api.ChargeStatus, error) { return c.status, nil }
func (c *feedInCharger) Enabled() (bool, error)            { return c.enabled, nil }
func (c *feedInCharger) MaxCurrent(v int64) error          { c.current = v; return nil }

func (c *feedInCharger) Enable(v bool) error {
	c.enabled = v
	// a disabled charger stops drawing, i.e. reports connected instead of charging
	if !v && c.status == api.StatusC {
		c.status = api.StatusB
	}
	return nil
}

var _ api.Charger = (*feedInCharger)(nil)

// newUpdatablePVLoadpoint returns a PV mode loadpoint that can run Update, unlike the
// bare newPVLoadpoint fixture used for the reservation logic itself
func newUpdatablePVLoadpoint(t *testing.T, title string, clck *clock.Mock, charger *feedInCharger, minCurrent, maxCurrent float64, phases, priority int) *Loadpoint {
	t.Helper()

	lp := newTestLoadpoint(title, clck)
	lp.charger = charger
	lp.chargeMeter = &Null{} // silence nil panics
	lp.chargeRater = &Null{} // silence nil panics
	lp.chargeTimer = &Null{} // silence nil panics
	lp.wakeUpTimer = NewTimer()
	lp.minCurrent = minCurrent
	lp.maxCurrent = maxCurrent
	lp.phases = phases
	lp.phasesConfigured = phases
	lp.priority = priority
	lp.status = charger.status

	attachListeners(t, lp) // sets Voltage = 230

	return lp
}

func TestPvChargeStarting(t *testing.T) {
	now := testNow

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
		// an expired timer means the loadpoint is not waiting to enable: it has been
		// paused and may resume immediately
		{"timer elapsed", newPVLoadpoint(0, api.ModePV, api.StatusB, false, elapsed), false},
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
	high := newPVLoadpoint(1, api.ModePV, api.StatusB, false, testNow)
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

// feedInSetup builds a site where the higher-priority loadpoint pauses for attractive
// feed-in while the lower-priority one wants the surplus:
//
//   - car charger: 3p, 6..11A, PV mode, priority 10, feed-in priority limit feedInLimit
//   - heating rod: 1p, 7..16A, PV mode, default priority 0
//
// The feed-in rate exceeds the car's limit, hence the car is paused to maximize feed-in.
func feedInSetup(t *testing.T, clck *clock.Mock, carStatus api.ChargeStatus, carEnabled bool) (*Site, *Loadpoint, *Loadpoint, api.Rates) {
	t.Helper()

	limit := feedInLimit
	car := newUpdatablePVLoadpoint(t, "car", clck, &feedInCharger{status: carStatus, enabled: carEnabled}, 6, 11, 3, 10)
	car.smartFeedInPriorityLimit = &limit
	car.limitSoc = 80

	heater := newUpdatablePVLoadpoint(t, "heater", clck, &feedInCharger{status: api.StatusB}, 7, 16, 1, 0)

	site := &Site{
		log:        util.NewLogger("site"),
		loadpoints: []*Loadpoint{car, heater},
	}

	feedin := api.Rates{{Start: testNow.Add(-time.Hour), End: testNow.Add(time.Hour), Value: feedInRate}}

	return site, car, heater, feedin
}

// TestReservedPVPowerSmartFeedInPause asserts that a loadpoint paused by its smart
// feed-in priority limit does not reserve surplus: it has been told to export instead
// of charge, so it will not consume that surplus.
func TestReservedPVPowerSmartFeedInPause(t *testing.T) {
	// the loadpoint may be idle or charging when the feed-in rate rises above its limit
	for _, tc := range []struct {
		name   string
		status api.ChargeStatus
		// charger state before the rate rise
		enabled bool
		soc     float64
	}{
		{"car idle", api.StatusB, false, 0},
		{"car charging", api.StatusC, true, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clck := newTestClock()
			site, car, heater, feedin := feedInSetup(t, clck, tc.status, tc.enabled)
			car.vehicleSoc = tc.soc

			// the pause holds across cycles, so must the absence of a reservation
			for i := range 4 {
				car.Update(testSurplus, 0, nil, feedin, false, false, 0, nil, nil, nil)

				if car.enabled {
					t.Fatalf("cycle %d: car must be paused by the feed-in limit", i)
				}

				if car.PvChargeStarting() {
					t.Errorf("cycle %d: paused car must not report starting up (pvTimer %v)", i, car.pvTimer)
				}

				if got := site.reservedPVPower(heater); got != 0 {
					t.Errorf("cycle %d: paused car reserves %.0fW from the heating rod, want 0W", i, got)
				}

				clck.Add(testEnableDelay / 2)
			}
		})
	}
}

// TestSmartFeedInPauseBlocksLowerPriority covers the consequence: while the car is
// paused by its feed-in limit, its reservation keeps the heating rod from starting.
func TestSmartFeedInPauseBlocksLowerPriority(t *testing.T) {
	clck := newTestClock()
	site, car, heater, feedin := feedInSetup(t, clck, api.StatusC, true)

	// pause the car- feed-in is more attractive than charging
	car.Update(testSurplus, 0, nil, feedin, false, false, 0, nil, nil, nil)
	if car.enabled {
		t.Fatal("car must be paused by the feed-in limit")
	}

	reserved := site.reservedPVPower(heater)
	t.Logf("paused car reserves %.0fW of the %.0fW surplus", reserved, -testSurplus)

	// the heating rod runs its enable sequence against the remaining surplus
	for range 4 {
		heater.Update(testSurplus+reserved, 0, nil, nil, false, false, 0, nil, nil, nil)
		clck.Add(testEnableDelay)
	}

	if !heater.enabled {
		t.Errorf("heating rod not started at %.0fW site power (%.0fW surplus less %.0fW reserved for the paused car)",
			testSurplus+reserved, -testSurplus, reserved)
	}
}

var _ loadpoint.API = (*Loadpoint)(nil)
