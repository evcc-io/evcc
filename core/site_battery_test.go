package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/mock"
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
		{api.StatusC, false, api.BatteryLocked, api.ModeNow},   // mode now, charging -> bat disabled
		{api.StatusB, false, api.BatteryNormal, api.ModeMinPV}, // mode minPV, not charging -> bat enabled
		{api.StatusC, false, api.BatteryNormal, api.ModeMinPV}, // mode minPV, charging -> bat enabled
		{api.StatusB, false, api.BatteryNormal, api.ModePV},    // mode PV, not charging -> bat enabled
		{api.StatusC, false, api.BatteryNormal, api.ModePV},    // mode PV, charging, no planner -> bat enabled
		{api.StatusC, true, api.BatteryLocked, api.ModePV},     // mode PV, charging, planner active -> bat disabled
	}

	log := util.NewLogger("foo")

	for _, tc := range tcs {

		batCtrl := mock.NewMockBatteryControl(ctrl)
		batCtrl.EXPECT().SetBatteryMode(tc.expBatMode).Times(1)

		s := &Site{
			// gridPower:    tc.grid,
			// pvPower:      tc.pv,
			// batteryPower: tc.battery,
			log:                     log,
			BatteryDischargeControl: true,
			batteryMeters:           []api.Meter{batCtrl},
		}

		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().GetStatus().Return(tc.chargeStatus).AnyTimes()
		lp.EXPECT().GetMode().Return(tc.mode).AnyTimes()
		lp.EXPECT().GetPlanActive().Return(tc.planActive).AnyTimes()

		loadpoints := []loadpoint.API{lp}
		err := s.UpdateBatteryMode(loadpoints)
		if err != nil {
			t.Errorf("error during UpdateBatteryDischarge, %s", err)
		}

		assert.Equal(t, tc.expBatMode, s.GetBatteryMode(), tc)
	}
}

func TestBatteryDischargeDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)

	batCtrl := mock.NewMockBatteryControl(ctrl)
	batCtrl.EXPECT().SetBatteryMode(gomock.Any()).Times(0)

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetStatus().Times(0)
	lp.EXPECT().GetMode().Times(0)
	lp.EXPECT().GetPlanActive().Times(0)
	loadpoints := []loadpoint.API{lp}

	s := &Site{
		// gridPower:    tc.grid,
		// pvPower:      tc.pv,
		// batteryPower: tc.battery,
		//log:           log,
		batteryMode:   api.BatteryNormal,
		batteryMeters: []api.Meter{batCtrl},
	}

	err := s.UpdateBatteryMode(loadpoints)

	if err != nil {
		t.Errorf("error during UpdateBatteryDischarge, %s", err)
	}

	assert.Equal(t, api.BatteryNormal, s.GetBatteryMode(), "disabled bat discharge control; battery modified nonetheless")
}
