package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRequiredExternalBatteryMode(t *testing.T) {
	for _, tc := range []struct {
		internal, ext, new api.BatteryMode
	}{
		{api.BatteryUnknown, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryUnknown, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryUnknown, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryNormal, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryNormal, api.BatteryUnknown}, // no change required
		{api.BatteryNormal, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryCharge, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryCharge, api.BatteryUnknown}, // no change required
	} {
		t.Logf("%+v", tc)

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{nil},
		}

		site.batteryMode = tc.internal
		site.batteryModeExternal = tc.ext

		mode := site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, tc.new.String(), mode.String(), "internal mode expected %s got %s", tc.new, mode)
	}
}

func TestExternalBatteryModeChange(t *testing.T) {
	ctrl := gomock.NewController(t)

	var bat api.Meter
	batCon := api.NewMockBatteryController(ctrl)

	bat = &struct {
		api.Meter
		api.BatteryController
	}{
		BatteryController: batCon,
	}

	for _, tc := range []struct {
		internal, ext, expected api.BatteryMode
	}{
		{api.BatteryUnknown, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryUnknown, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryUnknown, api.BatteryCharge, api.BatteryNormal},

		{api.BatteryNormal, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryNormal, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryNormal, api.BatteryCharge, api.BatteryNormal},

		{api.BatteryCharge, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryCharge, api.BatteryNormal},
	} {
		t.Logf("%+v", tc)

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		}

		// set initial state, internal mode may already be changed
		site.batteryMode = tc.internal
		site.batteryModeExternal = api.BatteryUnknown
		assert.True(t, site.batteryModeExternalTimer.IsZero())

		// 1. set required external mode
		site.SetBatteryModeExternal(tc.ext)
		assert.Equal(t, site.batteryModeExternal, tc.ext, "external mode expected %s got %s", tc.ext, site.batteryModeExternal)
		assert.Equal(t, site.batteryMode, tc.internal, "internal mode expected unchanged %s got %s", tc.ext, site.batteryMode)

		// 2. verify external mode applied to battery
		if tc.ext != api.BatteryUnknown {
			batCon.EXPECT().SetBatteryMode(tc.ext).Times(1)
		}
		site.updateBatteryMode(false, api.Rate{})
		ctrl.Finish()

		// 3. verify required external mode only applied once
		site.updateBatteryMode(false, api.Rate{})
		ctrl.Finish()

		// 4. verify timer expiry
		site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
		site.batteryModeWatchdogExpired()

		// mode reverted to unknown, timer still active
		assert.Equal(t, site.batteryModeExternal, api.BatteryUnknown)
		assert.False(t, site.batteryModeExternalTimer.IsZero())

		// battery switched back to normal mode
		batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
		site.updateBatteryMode(false, api.Rate{})
		ctrl.Finish()

		// timer disabled
		assert.True(t, site.batteryModeExternalTimer.IsZero())
	}
}
