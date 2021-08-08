package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/util"
	"github.com/golang/mock/gomock"
)

func TestVehicleDetectByStatus(t *testing.T) {
	ctrl := gomock.NewController(t)

	type vehicle struct {
		*mock.MockVehicle
		*mock.MockChargeState
	}

	v1 := &vehicle{mock.NewMockVehicle(ctrl), mock.NewMockChargeState(ctrl)}
	v2 := &vehicle{mock.NewMockVehicle(ctrl), mock.NewMockChargeState(ctrl)}

	type testcase struct {
		string
		v1, v2  api.ChargeStatus
		res     api.Vehicle
		prepare func(testcase)
	}
	tc := []testcase{
		{"A/A->0", api.StatusA, api.StatusA, nil, func(t testcase) {
			v1.MockChargeState.EXPECT().Status().Return(t.v1, nil)
			v2.MockChargeState.EXPECT().Status().Return(t.v2, nil)
		}},
		{"B/A->1", api.StatusB, api.StatusA, v1, func(t testcase) {
			v1.MockChargeState.EXPECT().Status().Return(t.v1, nil)
			v2.MockChargeState.EXPECT().Status().Return(t.v2, nil)
		}},
		{"A/B->2", api.StatusA, api.StatusB, v2, func(t testcase) {
			v1.MockChargeState.EXPECT().Status().Return(t.v1, nil)
			v2.MockChargeState.EXPECT().Status().Return(t.v2, nil)
		}},
		{"B/B->1", api.StatusB, api.StatusB, nil, func(t testcase) {
			v1.MockChargeState.EXPECT().Status().Return(t.v1, nil)
			v2.MockChargeState.EXPECT().Status().Return(t.v2, nil)
		}},
	}

	log := util.NewLogger("foo")
	vehicles := []api.Vehicle{v1, v2}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		tc.prepare(tc)

		v1.MockVehicle.EXPECT().Title().Return("v1")
		v2.MockVehicle.EXPECT().Title().Return("v2")

		lp := new(vehicleCoordinator)

		if res := lp.findActiveVehicleByStatus(log, vehicles); tc.res != res {
			t.Errorf("expected %v, got %v", tc.res, res)
		}
	}
}
