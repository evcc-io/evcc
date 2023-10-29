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
	lp := loadpoint.NewMockAPI(ctrl)

	tc := []struct {
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

	for _, tc := range tc {

		s := &Site{
			// gridPower:    tc.grid,
			// pvPower:      tc.pv,
			// batteryPower: tc.battery,
			log: log,
		}

		lp.EXPECT().GetStatus().AnyTimes().Return(tc.chargeStatus)
		lp.EXPECT().GetPlanActive().AnyTimes().Return(tc.planActive)

		loadpoints := []loadpoint.API{lp}
		err := s.UpdateBatteryMode(loadpoints)
		if err != nil {
			t.Errorf("error during UpdateBatteryDischarge, %s", err)
		}

		assert.Equal(t, tc.expBatMode, s.GetBatteryMode(), tc)
	}
}
