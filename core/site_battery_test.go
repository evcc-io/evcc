package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDetermineBatteryMode(t *testing.T) {
	ctrl := gomock.NewController(t)

	tcs := []struct {
		chargeStatus       api.ChargeStatus
		fastChargingActive bool
		smartCostActive    bool
		expBatMode         api.BatteryMode
	}{
		{api.StatusB, false, false, api.BatteryNormal}, // not charging | fast charge not active | smart cost not active -> bat normal
		{api.StatusB, true, false, api.BatteryNormal},  // not charging | fast charge active 	 | smart cost not active -> bat normal
		{api.StatusC, false, false, api.BatteryNormal}, // charging 	| fast charge not active | smart cost not active -> bat normal
		{api.StatusC, true, false, api.BatteryHold},    // charging 	| fast charge active 	 | smart cost not active -> bat hold
		{api.StatusB, false, true, api.BatteryNormal},  // not charging | fast charge not active | smart cost active	 -> bat normal
		{api.StatusB, true, true, api.BatteryNormal},   // not charging | fast charge active	 | smart cost active	 -> bat normal
		{api.StatusC, false, true, api.BatteryHold},    // charging 	| fast charge not active | smart cost active	 -> bat hold
		{api.StatusC, true, true, api.BatteryHold},     // charging 	| fast charge active	 | smart cost active	 -> bat hold
	}

	log := util.NewLogger("foo")

	for _, tc := range tcs {
		s := &Site{
			log: log,
		}

		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().GetStatus().Return(tc.chargeStatus).AnyTimes()
		lp.EXPECT().IsFastChargingActive().Return(tc.fastChargingActive).AnyTimes()

		loadpoints := []loadpoint.API{lp}

		mode := s.determineBatteryMode(loadpoints, tc.smartCostActive)
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
