package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/mock"
	"github.com/golang/mock/gomock"
)

type testCase struct {
	Mode                   api.ChargeMode
	MinCurrent, MaxCurrent int64
	ActualCurrent          int64
	CurrentPower           float64
	ExpectedCurrent        interface{}
}

func mockedLoadPoint(ctrl *gomock.Controller, tc testCase) *LoadPoint {
	cr := mock.NewMockCharger(ctrl)
	cr.EXPECT().
		Status().
		Return(api.StatusC, nil)
	cr.EXPECT().
		Enabled().
		Return(true, nil)
	cr.EXPECT().
		ActualCurrent().
		Return(tc.ActualCurrent, nil)

	m := mock.NewMockMeter(ctrl)
	m.EXPECT().
		CurrentPower().
		Return(tc.CurrentPower, nil)

	cc := mock.NewMockChargeController(ctrl)
	if expectedCurrent, ok := tc.ExpectedCurrent.(int64); ok {
		cc.EXPECT().
			MaxCurrent(gomock.Eq(expectedCurrent)).
			Return(nil)
	}

	lp := NewLoadPoint("lp1", &wrapper.CompositeCharger{
		Charger:          cr,
		ChargeController: cc,
	})

	lp.GridMeter = m
	lp.state.SetMode(tc.Mode)

	if tc.MinCurrent > 0 {
		lp.MinCurrent = tc.MinCurrent
	}
	if tc.MaxCurrent > 0 {
		lp.MaxCurrent = tc.MaxCurrent
	}

	return lp
}

func TestEVConnectedAndEnabledNowMode(t *testing.T) {
	cases := []testCase{
		testCase{api.ModeNow, 0, 0, 0, 0.0, 16},
		testCase{api.ModeNow, 0, 32, 0, 0.0, 32},
		testCase{api.ModeNow, 16, 32, 0, 0.0, 32},
	}

	for _, c := range cases {
		ctrl := gomock.NewController(t)

		lp := mockedLoadPoint(ctrl, c)
		lp.Update()

		ctrl.Finish()
	}
}

func TestEVConnectedAndEnabledPMinVMode(t *testing.T) {
	cases := []testCase{
		testCase{api.ModeMinPV, 0, 0, 5, 0.0, nil},
		testCase{api.ModeMinPV, 0, 0, 10, 1150.0, 5},
		testCase{api.ModeMinPV, 0, 0, 5, -1150.0, 10},
		testCase{api.ModeMinPV, 14, 0, 5, -1150.0, 14}, // 14A > 10A
	}

	for _, c := range cases {
		ctrl := gomock.NewController(t)

		lp := mockedLoadPoint(ctrl, c)
		lp.Update()

		ctrl.Finish()
	}
}

func TestEVConnectedAndEnabledPVMode(t *testing.T) {
	cases := []testCase{
		testCase{api.ModePV, 0, 0, 5, 0.0, nil},
		testCase{api.ModePV, 0, 0, 10, 1150.0, 5},
		testCase{api.ModePV, 0, 0, 5, -1150.0, 10},
		testCase{api.ModePV, 14, 0, 5, -1150.0, 0}, // 14A > 10A
	}

	for _, c := range cases {
		ctrl := gomock.NewController(t)

		lp := mockedLoadPoint(ctrl, c)
		lp.Update()

		ctrl.Finish()
	}
}
