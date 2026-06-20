package coordinator

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

func TestVehicleDetectByStatus(t *testing.T) {
	ctrl := gomock.NewController(t)

	type vehicle struct {
		*api.MockVehicle
		*api.MockChargeState
	}

	v1 := &vehicle{api.NewMockVehicle(ctrl), api.NewMockChargeState(ctrl)}
	v2 := &vehicle{api.NewMockVehicle(ctrl), api.NewMockChargeState(ctrl)}

	type testcase struct {
		string
		v1, v2 api.ChargeStatus
		res    api.Vehicle
	}
	tc := []testcase{
		{"A/A->0", api.StatusA, api.StatusA, nil},
		{"B/A->1", api.StatusB, api.StatusA, v1},
		{"B/A->1", api.StatusB, api.StatusA, v1},
		{"A/B->2", api.StatusA, api.StatusB, v2},
		{"A/B->2", api.StatusA, api.StatusB, v2},
		{"A/C->2", api.StatusA, api.StatusC, v2},
		{"A/C->2", api.StatusA, api.StatusC, v2},
		{"B/B->1", api.StatusB, api.StatusB, nil},
		{"B/C->1", api.StatusB, api.StatusC, v1},
		{"B/C->1", api.StatusB, api.StatusC, v1},
		{"C/B->2", api.StatusC, api.StatusB, v2},
		{"C/B->2", api.StatusC, api.StatusB, v2},
		{"C/C->1", api.StatusC, api.StatusC, nil},
	}

	log := util.NewLogger("foo")
	vehicles := []api.Vehicle{v1, v2}

	v1.MockVehicle.EXPECT().GetTitle().Return("v1").AnyTimes()
	v2.MockVehicle.EXPECT().GetTitle().Return("v2").AnyTimes()
	v1.MockVehicle.EXPECT().Identifiers().Return(nil).AnyTimes()
	v2.MockVehicle.EXPECT().Identifiers().Return([]string{"it's me"}).AnyTimes()
	v1.MockVehicle.EXPECT().Features().Return(nil).AnyTimes()
	v2.MockVehicle.EXPECT().Features().Return(nil).AnyTimes()

	var lp loadpoint.API
	c := New(log, vehicles)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		v1.MockChargeState.EXPECT().Status().Return(tc.v1, nil)
		v2.MockChargeState.EXPECT().Status().Return(tc.v2, nil)

		available := c.availableDetectibleVehicles(lp) // include id-able vehicles
		res := c.identifyVehicleByStatus(available, api.StatusB)
		if tc.res != res {
			t.Errorf("expected %v, got %v", tc.res, res)
		}

		if res != nil {
			c.acquire(lp, res)
		} else {
			c.release(lp, res)
		}
	}
}

// TestReleaseKeyedByOwner ensures that releasing a vehicle that has since been
// transferred to another loadpoint does not wipe the new owner's tracking.
//
// acquire() transfers a vehicle by deferring SetVehicle(nil) on the previous
// owner. That previous owner then releases the vehicle it still believes it
// holds - but the vehicle is now owned by the new loadpoint. A non-owner
// release must be a no-op, otherwise the vehicle ends up untracked and can be
// acquired by two loadpoints at once.
func TestReleaseKeyedByOwner(t *testing.T) {
	ctrl := gomock.NewController(t)

	v := api.NewMockVehicle(ctrl)
	v.EXPECT().GetTitle().Return("v").AnyTimes()

	c := New(util.NewLogger("foo"), []api.Vehicle{v})

	lp1 := loadpoint.NewMockAPI(ctrl)
	lp2 := loadpoint.NewMockAPI(ctrl)

	a1 := NewAdapter(lp1, c)
	a2 := NewAdapter(lp2, c)

	// model setActiveVehicle(nil): a loadpoint asked to drop its vehicle releases
	// the vehicle it still holds. lp1 holds v, so its SetVehicle(nil) releases v.
	lp1.EXPECT().SetVehicle(nil).Do(func(api.Vehicle) { a1.Release(v) }).AnyTimes()

	// lp1 acquires v
	a1.Acquire(v)
	if o := c.Owner(v); o != loadpoint.API(lp1) {
		t.Fatalf("expected v owned by lp1, got %v", o)
	}

	// lp2 takes v (cross-owner transfer). acquire defers lp1.SetVehicle(nil),
	// which releases v on lp1's behalf - this must not undo lp2's ownership.
	a2.Acquire(v)

	if o := c.Owner(v); o != loadpoint.API(lp2) {
		t.Errorf("v must stay owned by lp2 after transfer, got %v", o)
	}
}
