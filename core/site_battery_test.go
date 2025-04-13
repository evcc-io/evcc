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
			batteryMeters: []config.Device[api.Meter]{nil},
		}

		// set initial state, internal mode may already be changed
		site.batteryMode = tc.internal
		site.batteryModeExternal = api.BatteryUnknown
		assert.True(t, site.batteryModeExternalTimer.IsZero())

		// 1. set required external mode
		site.SetBatteryModeExternal(tc.ext)
		assert.Equal(t, site.batteryModeExternal, tc.ext, "external mode expected %s got %s", tc.ext, site.batteryModeExternal)
		assert.Equal(t, site.batteryMode, tc.internal, "internal mode expected unchanged %s got %s", tc.ext, site.batteryMode)

		// 2. verify required external mode is indicated (unless unknown)
		mode := site.requiredBatteryMode(false, api.Rate{})

		if tc.ext != api.BatteryUnknown {
			// timer started unless external mode is disabled by setting unknown
			assert.False(t, site.batteryModeExternalTimer.IsZero())

			// required mode should show external
			if tc.internal != tc.ext {
				assert.Equal(t, tc.ext, mode, "required mode expected %s got %s", tc.ext, mode)
			} else {
				assert.Equal(t, api.BatteryUnknown, mode, "required mode expected %s got %s", api.BatteryUnknown, mode)
			}
		} else {
			// required mode should be unknown unless internal mode has been changed
			if tc.internal != api.BatteryCharge {
				assert.Equal(t, api.BatteryUnknown, mode, "required mode expected %s got %s", api.BatteryUnknown, mode)
			}
		}

		// 3. apply required external mode (invoke applyBatteryMode, then SetBatteryMode if no error)
		if mode != api.BatteryUnknown {
			site.SetBatteryMode(mode)
			assert.Equal(t, mode, site.batteryMode, "internal mode expected %s got %s", mode, site.batteryMode)
		}

		// 4. required external mode should only be applied once
		mode = site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, api.BatteryUnknown, mode, "required mode should only be set once, expected %s got %s", api.BatteryUnknown, mode)

		// 5. timer expiry
		site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
		site.batteryModeWatchdogExpired()

		// mode reverted to unknown, timer still active
		assert.Equal(t, site.batteryModeExternal, api.BatteryUnknown)
		assert.False(t, site.batteryModeExternalTimer.IsZero())

		// switch battery back to normal mode
		mode = site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, tc.expected, mode, "external mode expected %s got %s", tc.expected, mode)

		// timer disabled
		site.SetBatteryMode(mode)
		assert.True(t, site.batteryModeExternalTimer.IsZero())
	}
}
