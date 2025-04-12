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
		{api.BatteryUnknown, api.BatteryHold, api.BatteryHold},
		{api.BatteryUnknown, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryNormal, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryNormal, api.BatteryHold, api.BatteryHold},
		{api.BatteryNormal, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryHold, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryHold, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryHold, api.BatteryHold, api.BatteryHold},
		{api.BatteryHold, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryCharge, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryHold, api.BatteryHold},
		{api.BatteryCharge, api.BatteryCharge, api.BatteryCharge},
	} {
		t.Logf("%+v", tc)

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{nil},
		}

		site.batteryMode = tc.internal
		// active external battery mode using setter
		site.SetBatteryModeExternal(tc.ext)

		// evaluate internal battery mode
		mode := site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, tc.new.String(), mode.String(), "external active, internal mode expected %s got %s", tc.new, mode)
	}
}

func TestExternalBatteryModeChange(t *testing.T) {
	for _, tc := range []struct {
		internal, ext, expired api.BatteryMode
	}{
		{api.BatteryUnknown, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryUnknown, api.BatteryNormal, api.BatteryUnknown},
		{api.BatteryUnknown, api.BatteryHold, api.BatteryNormal},
		{api.BatteryUnknown, api.BatteryCharge, api.BatteryNormal},

		{api.BatteryNormal, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryNormal, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryHold, api.BatteryNormal},
		{api.BatteryNormal, api.BatteryCharge, api.BatteryNormal},

		{api.BatteryHold, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryHold, api.BatteryNormal, api.BatteryUnknown},
		{api.BatteryHold, api.BatteryHold, api.BatteryNormal},
		{api.BatteryHold, api.BatteryCharge, api.BatteryNormal},

		{api.BatteryCharge, api.BatteryUnknown, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryNormal, api.BatteryUnknown},
		{api.BatteryCharge, api.BatteryHold, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryCharge, api.BatteryNormal},
	} {
		t.Logf("%+v", tc)

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{nil},
		}

		site.batteryMode = tc.internal

		// timer is initial
		assert.True(t, site.batteryModeExternalTimer.IsZero())

		// active external battery mode using setter
		site.SetBatteryModeExternal(tc.ext)

		// validate external battery mode
		assert.Equal(t, site.batteryModeExternal, tc.ext)

		// validate internal battery mode
		if tc.ext != api.BatteryUnknown {
			assert.Equal(t, site.GetBatteryMode(), tc.ext)
		}

		// timer check
		if tc.ext != api.BatteryUnknown {
			// external modes normal/hold/charge - timer active
			assert.False(t, site.batteryModeExternalTimer.IsZero())
			assert.False(t, site.batteryModeWatchdogExpired())
			// expire timer
			site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
		} else {
			// external mode unknown - timer expired forcefully before watchdog activity
			assert.True(t, site.batteryModeExternalTimer.IsZero())
		}

		// check for expired timer (independent of external battery mode)
		assert.True(t, site.batteryModeWatchdogExpired())

		// wait for expiration watchdog to handle changed timer (looped every second, for safety wait longer)
		time.Sleep(2 * time.Second)

		// mode reverted to unknown, timer inactive
		assert.Equal(t, site.batteryModeExternal, api.BatteryUnknown)
		assert.True(t, site.batteryModeExternalTimer.IsZero())

		// evaluate internal battery mode
		mode := site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, tc.expired.String(), mode.String(), "external expired, internal mode expected %s got %s", tc.expired, mode)

		// on valid battery mode
		if mode != api.BatteryUnknown {
			site.SetBatteryMode(mode)
			// timer sill disabled
			assert.True(t, site.batteryModeExternalTimer.IsZero())
		}

		if tc.internal != api.BatteryUnknown || tc.ext != api.BatteryUnknown {
			// check internal battery mode after changes and expiration: ensures valid batteryMode
			assert.Equal(t, site.GetBatteryMode(), api.BatteryNormal)
		}
	}
}
