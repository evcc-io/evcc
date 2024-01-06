package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineBatteryMode(t *testing.T) {
	ctrl := gomock.NewController(t)

	tcs := []struct {
		chargeStatus api.ChargeStatus
		planActive   bool
		expBatMode   api.BatteryMode
		mode         api.ChargeMode
	}{
		{api.StatusB, false, api.BatteryNormal, api.ModeOff},   // mode off -> bat normal
		{api.StatusB, false, api.BatteryNormal, api.ModeNow},   // mode now, not charging -> bat normal
		{api.StatusC, false, api.BatteryHold, api.ModeNow},     // mode now, charging -> bat hold
		{api.StatusB, false, api.BatteryNormal, api.ModeMinPV}, // mode minPV, not charging -> bat normal
		{api.StatusC, false, api.BatteryNormal, api.ModeMinPV}, // mode minPV, charging -> bat normal
		{api.StatusB, false, api.BatteryNormal, api.ModePV},    // mode PV, not charging -> bat normal
		{api.StatusC, false, api.BatteryNormal, api.ModePV},    // mode PV, charging, no planner -> bat normal
		{api.StatusC, true, api.BatteryHold, api.ModePV},       // mode PV, charging, planner active -> bat hold
	}

	log := util.NewLogger("foo")

	for _, tc := range tcs {
		s := &Site{
			log: log,
		}

		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().GetStatus().Return(tc.chargeStatus).AnyTimes()
		lp.EXPECT().GetMode().Return(tc.mode).AnyTimes()
		lp.EXPECT().GetPlanActive().Return(tc.planActive).AnyTimes()

		loadpoints := []loadpoint.API{lp}

		mode := s.determineBatteryMode(loadpoints, false)
		assert.Equal(t, tc.expBatMode, mode, tc)
	}
}

func TestUpdateBatteryMode(t *testing.T) {
	expBatMode := api.BatteryHold

	ctrl := gomock.NewController(t)

	batCtrl := struct {
		*api.MockBatteryController
		*api.MockMeter
	}{
		api.NewMockBatteryController(ctrl),
		api.NewMockMeter(ctrl),
	}
	batCtrl.MockBatteryController.EXPECT().SetBatteryMode(expBatMode).Times(1)

	s := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []api.Meter{batCtrl},
		batteryMode:   api.BatteryNormal,
	}

	err := s.updateBatteryMode(expBatMode)
	require.NoError(t, err)
	assert.Equal(t, expBatMode, s.GetBatteryMode())
}
