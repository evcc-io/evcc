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
		v1, v2                      api.ChargeStatus
		v2ExcludedFromAutoDiscovery bool
		res                         api.Vehicle
	}
	tc := []testcase{
		{"A/A, v2 not excluded from AutoDiscovery ->0", api.StatusA, api.StatusA, false, nil},
		{"B/A, v2 not excluded from AutoDiscovery ->1", api.StatusB, api.StatusA, false, v1},
		{"B/A, v2 not excluded from AutoDiscovery ->1", api.StatusB, api.StatusA, false, v1},
		{"A/B, v2 not excluded from AutoDiscovery ->2", api.StatusA, api.StatusB, false, v2},
		{"A/B, v2 not excluded from AutoDiscovery ->2", api.StatusA, api.StatusB, false, v2},
		{"A/B, v2     excluded from AutoDiscovery ->2", api.StatusA, api.StatusB, true, nil},
		{"A/B, v2     excluded from AutoDiscovery ->2", api.StatusA, api.StatusB, true, nil},
		{"A/C, v2 not excluded from AutoDiscovery ->2", api.StatusA, api.StatusC, false, v2},
		{"A/C, v2 not excluded from AutoDiscovery ->2", api.StatusA, api.StatusC, false, v2},
		{"B/B, v2 not excluded from AutoDiscovery ->1", api.StatusB, api.StatusB, false, nil},
		{"B/B, v2     excluded from AutoDiscovery ->1", api.StatusB, api.StatusB, true, v1},
		{"B/C, v2 not excluded from AutoDiscovery ->1", api.StatusB, api.StatusC, false, v1},
		{"B/C, v2 not excluded from AutoDiscovery ->1", api.StatusB, api.StatusC, false, v1},
		{"C/B, v2 not excluded from AutoDiscovery ->2", api.StatusC, api.StatusB, false, v2},
		{"C/B, v2 not excluded from AutoDiscovery ->2", api.StatusC, api.StatusB, false, v2},
		{"C/C, v2 not excluded from AutoDiscovery ->1", api.StatusC, api.StatusC, false, nil},
	}

	log := util.NewLogger("foo")
	vehicles := []api.Vehicle{v1, v2}

	v1.MockVehicle.EXPECT().GetTitle().Return("v1").AnyTimes()
	v2.MockVehicle.EXPECT().GetTitle().Return("v2").AnyTimes()
	v1.MockVehicle.EXPECT().Identifiers().Return(nil).AnyTimes()
	v2.MockVehicle.EXPECT().Identifiers().Return([]string{"it's me"}).AnyTimes()
	v1.MockVehicle.EXPECT().Features().Return(nil).AnyTimes()

	var lp loadpoint.API
	c := New(log, vehicles)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		v1.MockChargeState.EXPECT().Status().Return(tc.v1, nil)
		if tc.v2ExcludedFromAutoDiscovery {
			v2.MockVehicle.EXPECT().Features().Return([]api.Feature{api.AutodetectDisabled})
		} else {
			v2.MockVehicle.EXPECT().Features().Return(nil)
			v2.MockChargeState.EXPECT().Status().Return(tc.v2, nil)
		}

		available := c.availableDetectibleVehicles(lp) // include id-able vehicles
		res := c.identifyVehicleByStatus(available, api.StatusB)
		if tc.res != res {
			t.Errorf("expected %v, got %v", tc.res, res)
		}

		if res != nil {
			c.acquire(lp, res)
		} else {
			c.release(res)
		}
	}
}
