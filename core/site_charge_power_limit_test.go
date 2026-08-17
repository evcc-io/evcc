package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestApplyBatteryChargePowerLimit(t *testing.T) {
	for _, tc := range []struct {
		mode      api.BatteryMode
		limit     *float64
		wantApply *float64
	}{
		{api.BatteryUnknown, new(1000.0), new(1000.0)},
		{api.BatteryNormal, new(1000.0), new(1000.0)},
		{api.BatteryCharge, new(1000.0), new(1000.0)},
		{api.BatteryHold, new(1000.0), nil},       // hold means don't charge
		{api.BatteryHoldCharge, new(1000.0), nil}, // holdcharge means don't charge
	} {
		t.Logf("%+v", tc)

		ctrl := gomock.NewController(t)

		limiter := api.NewMockBatteryChargePowerLimiter(ctrl)
		var bat api.Meter = &struct {
			api.Meter
			api.BatteryChargePowerLimiter
		}{
			BatteryChargePowerLimiter: limiter,
		}

		site := &Site{
			log:                             util.NewLogger("foo"),
			batteryMeters:                   []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
			batteryMode:                     tc.mode,
			batteryChargePowerLimitExternal: tc.limit,
			batteryChargePowerLimit:         new(-1.0), // sentinel: force a write regardless of target
		}

		limiter.EXPECT().SetBatteryChargePowerLimit(tc.wantApply).Times(1)

		site.updateBatteryChargePowerLimit()
		assert.Equal(t, ptrDeref(tc.wantApply), ptrDeref(site.GetBatteryChargePowerLimit()))

		ctrl.Finish()
	}
}

func ptrDeref(f *float64) float64 {
	if f == nil {
		return -1
	}
	return *f
}

// TestBatteryChargePowerLimitNoRedundantWrite guards that the limiter is only
// invoked when the effective (mode-scoped) limit actually changes.
func TestBatteryChargePowerLimitNoRedundantWrite(t *testing.T) {
	ctrl := gomock.NewController(t)

	limiter := api.NewMockBatteryChargePowerLimiter(ctrl)
	var bat api.Meter = &struct {
		api.Meter
		api.BatteryChargePowerLimiter
	}{
		BatteryChargePowerLimiter: limiter,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		batteryMode:   api.BatteryNormal,
	}

	limiter.EXPECT().SetBatteryChargePowerLimit(new(500.0)).Times(1)
	site.batteryChargePowerLimitExternal = new(500.0)
	site.updateBatteryChargePowerLimit()

	// same limit, same mode -> no additional call
	site.updateBatteryChargePowerLimit()

	// mode change to Hold suppresses the limit -> one call with nil
	limiter.EXPECT().SetBatteryChargePowerLimit((*float64)(nil)).Times(1)
	site.batteryMode = api.BatteryHold
	site.updateBatteryChargePowerLimit()

	// staying in Hold -> no additional call
	site.updateBatteryChargePowerLimit()

	ctrl.Finish()
}

func TestExternalBatteryChargePowerLimitWatchdog(t *testing.T) {
	ctrl := gomock.NewController(t)

	limiter := api.NewMockBatteryChargePowerLimiter(ctrl)
	var bat api.Meter = &struct {
		api.Meter
		api.BatteryChargePowerLimiter
	}{
		BatteryChargePowerLimiter: limiter,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		batteryMode:   api.BatteryNormal,
	}

	settings.SetBool(keys.Experimental, true)
	defer settings.SetBool(keys.Experimental, false)

	assert.NoError(t, site.SetBatteryChargePowerLimitExternal(new(500.0)))
	assert.False(t, site.batteryChargePowerLimitExternalTimer.IsZero())

	// establish the applied state before expiring the watchdog
	limiter.EXPECT().SetBatteryChargePowerLimit(new(500.0)).Times(1)
	site.updateBatteryChargePowerLimit()

	// expire the watchdog
	site.batteryChargePowerLimitExternalTimer = site.batteryChargePowerLimitExternalTimer.Add(-time.Hour)
	assert.True(t, site.batteryChargePowerLimitWatchdogExpired())

	assert.Nil(t, site.GetBatteryChargePowerLimitExternal())
	assert.True(t, site.batteryChargePowerLimitExternalTimer.IsZero())

	// applying the released state issues a release write
	limiter.EXPECT().SetBatteryChargePowerLimit((*float64)(nil)).Times(1)
	site.updateBatteryChargePowerLimit()

	ctrl.Finish()
}

func TestSetBatteryChargePowerLimitExternalRejectsNegative(t *testing.T) {
	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{nil},
	}

	assert.Error(t, site.SetBatteryChargePowerLimitExternal(new(-1.0)))
}

// TestBatteryChargePowerLimitRequiresExperimental guards that setting a limit is gated
// behind the experimental flag, while releasing it stays possible regardless so an
// already-applied limit can always be cleared.
func TestBatteryChargePowerLimitRequiresExperimental(t *testing.T) {
	ctrl := gomock.NewController(t)

	limiter := api.NewMockBatteryChargePowerLimiter(ctrl)
	var bat api.Meter = &struct {
		api.Meter
		api.BatteryChargePowerLimiter
	}{
		BatteryChargePowerLimiter: limiter,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		batteryMode:   api.BatteryNormal,
	}

	settings.SetBool(keys.Experimental, false)

	assert.ErrorIs(t, site.SetBatteryChargePowerLimitExternal(new(500.0)), ErrExperimentalNotEnabled)
	assert.Nil(t, site.GetBatteryChargePowerLimitExternal())

	// releasing is always allowed
	assert.NoError(t, site.SetBatteryChargePowerLimitExternal(nil))

	settings.SetBool(keys.Experimental, true)
	defer settings.SetBool(keys.Experimental, false)

	assert.NoError(t, site.SetBatteryChargePowerLimitExternal(new(500.0)))
	assert.Equal(t, 500.0, *site.GetBatteryChargePowerLimitExternal())

	ctrl.Finish()
}
