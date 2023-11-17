package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBatteryDischarge(t *testing.T) {
	ctrl := gomock.NewController(t)

	tcs := []struct {
		chargeStatus api.ChargeStatus
		planActive   bool
		expBatMode   api.BatteryMode
		mode         api.ChargeMode
	}{
		{api.StatusB, false, api.BatteryNormal, api.ModeOff},   // mode off -> bat enabled
		{api.StatusB, false, api.BatteryNormal, api.ModeNow},   // mode now, not charging -> bat enabled
		{api.StatusC, false, api.BatteryHold, api.ModeNow},     // mode now, charging -> bat disabled
		{api.StatusB, false, api.BatteryNormal, api.ModeMinPV}, // mode minPV, not charging -> bat enabled
		{api.StatusC, false, api.BatteryNormal, api.ModeMinPV}, // mode minPV, charging -> bat enabled
		{api.StatusB, false, api.BatteryNormal, api.ModePV},    // mode PV, not charging -> bat enabled
		{api.StatusC, false, api.BatteryNormal, api.ModePV},    // mode PV, charging, no planner -> bat enabled
		{api.StatusC, true, api.BatteryHold, api.ModePV},       // mode PV, charging, planner active -> bat disabled
	}

	log := util.NewLogger("foo")

	for _, tc := range tcs {
		batCtrl := struct {
			*api.MockBatteryController
			*api.MockMeter
		}{
			api.NewMockBatteryController(ctrl),
			api.NewMockMeter(ctrl),
		}
		batCtrl.MockBatteryController.EXPECT().SetBatteryMode(tc.expBatMode).Times(1)

		s := &Site{
			log:                     log,
			BatteryDischargeControl: true,
			batteryMeters:           []api.Meter{batCtrl},
		}

		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().GetStatus().Return(tc.chargeStatus).AnyTimes()
		lp.EXPECT().GetMode().Return(tc.mode).AnyTimes()
		lp.EXPECT().GetPlanActive().Return(tc.planActive).AnyTimes()

		loadpoints := []loadpoint.API{lp}
		s.updateBatteryMode(loadpoints)
		assert.Equal(t, tc.expBatMode, s.getBatteryMode(), tc)
	}
}

// test that BatteryControllers are only called if batterymode changes
func TestBatteryModeNoUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)

	batCtrl := struct {
		*api.MockBatteryController
		*api.MockMeter
	}{
		api.NewMockBatteryController(ctrl),
		api.NewMockMeter(ctrl),
	}
	batCtrl.MockBatteryController.EXPECT().SetBatteryMode(api.BatteryHold).Times(1)

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetStatus().Return(api.StatusC).Times(2)
	lp.EXPECT().GetMode().Return(api.ModeNow).Times(2)
	lp.EXPECT().GetPlanActive().Times(0)
	loadpoints := []loadpoint.API{lp}

	s := &Site{
		batteryMode:             api.BatteryNormal,
		batteryMeters:           []api.Meter{batCtrl},
		BatteryDischargeControl: true,
		log:                     util.NewLogger("foo"),
	}

	s.updateBatteryMode(loadpoints) // first call should call BatteryController
	s.updateBatteryMode(loadpoints) // this one should not

	// adjust mocks to simulate charge stop, should cause batMode udpate
	lp.EXPECT().GetStatus().Return(api.StatusB).Times(1)
	lp.EXPECT().GetMode().Return(api.ModeNow).Times(0)
	batCtrl.MockBatteryController.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)

	s.updateBatteryMode(loadpoints) // this one should have updated again
}
