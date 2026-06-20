package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// chargeStateVehicle is a vehicle that also reports a charge state, so it is
// eligible for status-based detection by the coordinator.
type chargeStateVehicle struct {
	*api.MockVehicle
	*api.MockChargeState
}

func newChargeStateVehicle(ctrl *gomock.Controller, title string, status api.ChargeStatus) *chargeStateVehicle {
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().GetTitle().Return(title).AnyTimes()
	v.EXPECT().Icon().Return("").AnyTimes()
	v.EXPECT().Capacity().AnyTimes()
	v.EXPECT().Phases().AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()
	v.EXPECT().Identifiers().Return(nil).AnyTimes()
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Soc().Return(0.0, nil).AnyTimes()

	cs := api.NewMockChargeState(ctrl)
	cs.EXPECT().Status().Return(status, nil).AnyTimes()

	return &chargeStateVehicle{v, cs}
}

// fixedStatusCharger is a charger reporting a constant status.
type fixedStatusCharger struct{ status api.ChargeStatus }

func (c *fixedStatusCharger) Status() (api.ChargeStatus, error) { return c.status, nil }
func (c *fixedStatusCharger) Enabled() (bool, error)            { return true, nil }
func (c *fixedStatusCharger) Enable(bool) error                 { return nil }
func (c *fixedStatusCharger) MaxCurrent(int64) error            { return nil }

func newCoordinatedLoadpoint(t *testing.T, clck clock.Clock, charger api.Charger, dflt api.Vehicle) *Loadpoint {
	t.Helper()

	lp := &Loadpoint{
		log:            util.NewLogger("lp"),
		bus:            evbus.New(),
		clock:          clck,
		charger:        charger,
		chargeMeter:    &Null{},
		chargeRater:    &Null{},
		chargeTimer:    &Null{},
		wakeUpTimer:    NewTimer(),
		minCurrent:     minA,
		maxCurrent:     maxA,
		phases:         1,
		mode:           api.ModeNow,
		defaultVehicle: dflt,
	}

	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	return lp
}

// TestVehicleCoordinatorNoDoubleAssign is a regression test for the coordinator
// losing its mutual-exclusion invariant.
//
// Setup mirrors the live two-wallbox/two-Tesla report:
//
//	LP1: charger status C, configured default vehicle Luna   (Luna charging,   status C)
//	LP2: charger status B, configured default vehicle Calypso (Calypso idle,    status B)
//
// When a loadpoint transiently picks up the other loadpoint's vehicle via
// status detection ("LP2: Calypso -> Luna" in the live log), the previous
// owner's deferred SetVehicle(nil) used to wipe the freshly transferred
// ownership entry (release was not keyed by owner). The vehicle then became
// untracked, so the next default re-assignment on the original loadpoint
// re-acquired it without releasing the other loadpoint - leaving the same
// vehicle active on BOTH loadpoints and the configured default silently lost.
//
// The invariants asserted after every step:
//  1. no single vehicle is active on two loadpoints at once, and
//  2. a loadpoint never holds a vehicle the coordinator says is owned by a
//     different loadpoint (i.e. a configured default is never replaced by a
//     vehicle owned by another loadpoint).
func TestVehicleCoordinatorNoDoubleAssign(t *testing.T) {
	ctrl := gomock.NewController(t)
	clck := clock.NewMock()

	luna := newChargeStateVehicle(ctrl, "Luna", api.StatusC)       // charging
	calypso := newChargeStateVehicle(ctrl, "Calypso", api.StatusB) // connected, idle

	coord := coordinator.New(util.NewLogger("coord"), []api.Vehicle{luna, calypso})

	lp1 := newCoordinatedLoadpoint(t, clck, &fixedStatusCharger{status: api.StatusC}, luna)
	lp2 := newCoordinatedLoadpoint(t, clck, &fixedStatusCharger{status: api.StatusB}, calypso)
	lp1.coordinator = coordinator.NewAdapter(lp1, coord)
	lp2.coordinator = coordinator.NewAdapter(lp2, coord)

	lps := map[*Loadpoint]string{lp1: "LP1", lp2: "LP2"}

	ownerName := func(o loadpoint.API) string {
		switch o {
		case nil:
			return "<untracked>"
		case loadpoint.API(lp1):
			return "LP1"
		case loadpoint.API(lp2):
			return "LP2"
		}
		return "?"
	}

	assertInvariants := func(label string) {
		// (1) no shared vehicle across loadpoints
		if v := lp1.GetVehicle(); v != nil && v == lp2.GetVehicle() {
			t.Errorf("%s: vehicle %q active on both loadpoints at once", label, v.GetTitle())
		}
		// (2) every held vehicle is owned by the holding loadpoint
		for lp, name := range lps {
			if v := lp.GetVehicle(); v != nil {
				if o := coord.Owner(v); o != loadpoint.API(lp) {
					t.Errorf("%s: %s holds %q but coordinator owner is %s", label, name, v.GetTitle(), ownerName(o))
				}
			}
		}
	}

	// defaults assigned on connect: Luna -> LP1, Calypso -> LP2
	lp1.setActiveVehicle(luna)
	lp2.setActiveVehicle(calypso)
	assertInvariants("after defaults assigned")

	// transient: LP2 picks up Luna via status detection (live log: "LP2: Calypso -> Luna").
	// acquire() transfers Luna to LP2 and defers LP1.SetVehicle(nil); the previous
	// owner LP1 must not wipe LP2's ownership when it releases its now-stale vehicle.
	lp2.setActiveVehicle(luna)
	assertInvariants("after LP2 transiently acquires Luna")

	// LP1 re-asserts its configured default Luna on the next cycle (live log:
	// "LP1: unknown -> Luna"). This must transfer Luna away from LP2, not clone it.
	for cycle := 0; cycle < 3; cycle++ {
		lp1.setActiveVehicle(lp1.defaultVehicle)
		assertInvariants("after LP1 re-asserts default Luna")

		// LP2 only re-detects while it has no vehicle; mirrors the guarded
		// identifyVehicleByStatus call in the update loop.
		if lp2.vehicleUnidentified() {
			lp2.identifyVehicleByStatus()
		}
		assertInvariants("after LP2 status detection")
	}
}
