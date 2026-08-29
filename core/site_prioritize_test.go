package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
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
		// elapsed means a delay was skipped, e.g. by a feed-in pause, not an enable pending
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

			car := newPVLoadpoint(1, api.ModePV, tc.status, tc.enabled, time.Time{})
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

			low := newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{})

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

// newPrioritySite ranks the given loadpoints by the given strategy and deadband
func newPrioritySite(strategy api.PriorityStrategy, hysteresis int, lps ...*Loadpoint) *Site {
	return &Site{
		log:                util.NewLogger("site"),
		loadpoints:         lps,
		PriorityStrategy:   strategy,
		PriorityHysteresis: hysteresis,
	}
}

// startingPVLoadpoint returns a PV loadpoint with the given soc and an enable timer running
func startingPVLoadpoint(prio int, soc float64) *Loadpoint {
	lp := newPVLoadpoint(prio, api.ModePV, api.StatusB, false, clock.NewMock().Now())
	lp.vehicleSoc = soc
	return lp
}

// TestReservedPVPowerWithinTier asserts that the priority strategy orders the enable race
// inside a tier: the emptier car starts against the surplus, the fuller one defers to it.
func TestReservedPVPowerWithinTier(t *testing.T) {
	Voltage = 230

	// both claim the same surplus at the same priority tier
	emptier := startingPVLoadpoint(0, 20)
	fuller := startingPVLoadpoint(0, 80)

	for _, lp := range []*Loadpoint{emptier, fuller} {
		if !lp.PvChargeStarting() {
			t.Fatalf("loadpoint at soc %.0f must be starting up", lp.GetSoc())
		}
	}

	site := newPrioritySite(api.PrioritySoc, 0, emptier, fuller)

	if got := site.reservedPVPower(emptier); got != 0 {
		t.Errorf("emptier: want 0, got %.0f", got)
	}
	if got, want := site.reservedPVPower(fuller), emptier.EffectiveMaxPower(); got != want {
		t.Errorf("fuller: want %.0f, got %.0f", want, got)
	}
}

// TestReservedPVPowerStrategyNone asserts the score comparison is a strict superset of the
// tier comparison: without a strategy the fraction is 0, hence only the tier decides.
func TestReservedPVPowerStrategyNone(t *testing.T) {
	Voltage = 230

	// same tier, different soc: unordered, both race for the surplus
	emptier := startingPVLoadpoint(0, 20)
	fuller := startingPVLoadpoint(0, 80)

	site := newPrioritySite(api.PriorityNone, 0, emptier, fuller)

	if got := site.reservedPVPower(emptier); got != 0 {
		t.Errorf("same tier, emptier: want 0, got %.0f", got)
	}
	if got := site.reservedPVPower(fuller); got != 0 {
		t.Errorf("same tier, fuller: want 0, got %.0f", got)
	}

	// an explicit tier still wins, here against the soc the strategy would have ranked by
	fuller.priority = 1

	if got, want := site.reservedPVPower(emptier), fuller.EffectiveMaxPower(); got != want {
		t.Errorf("cross tier, low: want %.0f, got %.0f", want, got)
	}
	if got := site.reservedPVPower(fuller); got != 0 {
		t.Errorf("cross tier, high: want 0, got %.0f", got)
	}
}

// TestReservedPVPowerHysteresis asserts the deadband applies to the enable race as well:
// a score gap inside the band leaves both racing, exactly as an equal tier does.
func TestReservedPVPowerHysteresis(t *testing.T) {
	Voltage = 230

	emptier := startingPVLoadpoint(0, 20)
	fuller := startingPVLoadpoint(0, 25)

	// 10 soc-% deadband on the percent basis
	site := newPrioritySite(api.PrioritySoc, 10, emptier, fuller)

	// 5 points apart: inside the band, neither defers
	if got := site.reservedPVPower(fuller); got != 0 {
		t.Errorf("within band, fuller: want 0, got %.0f", got)
	}
	if got := site.reservedPVPower(emptier); got != 0 {
		t.Errorf("within band, emptier: want 0, got %.0f", got)
	}

	// 25 points apart: beyond the band, the fuller one defers
	fuller.vehicleSoc = 45

	if got, want := site.reservedPVPower(fuller), emptier.EffectiveMaxPower(); got != want {
		t.Errorf("beyond band, fuller: want %.0f, got %.0f", want, got)
	}
	if got := site.reservedPVPower(emptier); got != 0 {
		t.Errorf("beyond band, emptier: want 0, got %.0f", got)
	}
}

// TestReservedPVPowerAcrossTiers asserts the deadband sub-orders within a tier only: an
// explicit priority wins even against a band wide enough to swallow a whole tier.
func TestReservedPVPowerAcrossTiers(t *testing.T) {
	Voltage = 230

	ctrl := gomock.NewController(t)
	newVehicle := func() api.Vehicle {
		v := api.NewMockVehicle(ctrl)
		v.EXPECT().Capacity().Return(40.0).AnyTimes()
		v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
		v.EXPECT().Features().Return(nil).AnyTimes()
		v.EXPECT().Phases().Return(0).AnyTimes()
		return v
	}

	// the fuller car outranks by tier, the emptier one by strategy
	low := startingPVLoadpoint(0, 20)
	low.vehicle = newVehicle()
	high := startingPVLoadpoint(1, 80)
	high.vehicle = newVehicle()

	// 99 kWh against a 40 kWh reference: a band wider than one tier
	site := newPrioritySite(api.PrioritySoc, 99, low, high)
	site.PriorityBasis = api.PriorityBasisEnergy

	if _, ref := site.EffectivePriorityScoring(); ref != 40 {
		t.Fatalf("reference: want 40, got %.0f", ref)
	}

	if got, want := site.reservedPVPower(low), high.EffectiveMaxPower(); got != want {
		t.Errorf("low: want %.0f, got %.0f", want, got)
	}
	if got := site.reservedPVPower(high); got != 0 {
		t.Errorf("high: want 0, got %.0f", got)
	}
}

// TestReservedPVPowerHeating asserts a same-tier pair involving heating is left alone:
// heating aliases temperature as soc and carries no score comparable to a car's.
func TestReservedPVPowerHeating(t *testing.T) {
	Voltage = 230

	ctrl := gomock.NewController(t)
	describer := api.NewMockFeatureDescriber(ctrl)
	describer.EXPECT().Features().Return([]api.Feature{api.Heating}).AnyTimes()

	car := startingPVLoadpoint(0, 20)

	heater := newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{})
	heater.vehicleSoc = 55 // temperature, not a charge level
	heater.charger = struct {
		api.Charger
		api.FeatureDescriber
	}{
		Charger:          api.NewMockCharger(ctrl),
		FeatureDescriber: describer,
	}

	site := newPrioritySite(api.PrioritySoc, 0, heater, car)

	if got := site.reservedPVPower(heater); got != 0 {
		t.Errorf("heater: want 0, got %.0f", got)
	}

	// control: the same soc without the heating feature is comparable and does defer
	plain := newPVLoadpoint(0, api.ModePV, api.StatusB, false, time.Time{})
	plain.vehicleSoc = 55
	site.loadpoints = []*Loadpoint{plain, car}

	if got, want := site.reservedPVPower(plain), car.EffectiveMaxPower(); got != want {
		t.Errorf("plain: want %.0f, got %.0f", want, got)
	}
}

var _ loadpoint.API = (*Loadpoint)(nil)
