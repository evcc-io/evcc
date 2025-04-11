package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

func TestExternalBatteryMode(t *testing.T) {
	for _, tc := range []struct {
		internal, ext, new api.BatteryMode
	}{
		{api.BatteryUnknown, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryUnknown, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryUnknown, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryNormal, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryNormal, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryCharge, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryCharge, api.BatteryCharge},
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
	for _, tc := range []struct {
		internal, ext, expired api.BatteryMode
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
			batteryMeters: []config.Device[api.Meter]{nil},
		}

		site.batteryMode = tc.internal

		assert.True(t, site.batteryModeExternalTimer.IsZero())
		site.SetBatteryModeExternal(tc.ext)

		// timer started
		if tc.ext != api.BatteryUnknown {
			assert.False(t, site.batteryModeExternalTimer.IsZero())
		}

		// expire timer
		site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
		site.batteryModeWatchdogExpired()

		// mode reverted to unknown, timer still active
		assert.Equal(t, site.batteryModeExternal, api.BatteryUnknown)
		assert.False(t, site.batteryModeExternalTimer.IsZero())

		mode := site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, tc.expired.String(), mode.String(), "external mode expected %s got %s", tc.expired, mode)

		// timer disabled
		site.SetBatteryMode(mode)
		assert.True(t, site.batteryModeExternalTimer.IsZero())
	}
}
