package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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

	err := s.applyBatteryMode(expBatMode)
	require.NoError(t, err)
	assert.Equal(t, expBatMode, s.GetBatteryMode())
}
