package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestApplyBatteryMode(t *testing.T) {
	for _, tc := range []struct {
		internal, expected api.BatteryMode
	}{
		{api.BatteryUnknown, api.BatteryUnknown}, // no change required
		{api.BatteryNormal, api.BatteryUnknown},  // no change required
		{api.BatteryHold, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryNormal},
	} {
		t.Logf("%+v", tc)

		ctrl := gomock.NewController(t)

		var bat api.Meter
		batCon := api.NewMockBatteryController(ctrl)

		bat = &struct {
			api.Meter
			api.BatteryController
		}{
			BatteryController: batCon,
		}

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
			batteryMode:   tc.internal,
		}

		// verify mode applied to battery
		if tc.expected != api.BatteryUnknown {
			batCon.EXPECT().SetBatteryMode(tc.expected).Times(1)
		}
		site.updateBatteryMode(false, api.Rate{})

		if tc.internal != api.BatteryNormal {
			assert.Equal(t, tc.expected, site.batteryMode)
		}

		ctrl.Finish()
	}
}

func TestRequiredExternalBatteryMode(t *testing.T) {
	for _, tc := range []struct {
		internal, external, new api.BatteryMode
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
		site.batteryModeExternal = tc.external

		mode := site.requiredBatteryMode(false, api.Rate{})
		assert.Equal(t, tc.new.String(), mode.String(), "internal mode expected %s got %s", tc.new, mode)
	}
}

func TestExternalBatteryModeChange(t *testing.T) {
	for _, tc := range []struct {
		internal, external, expected api.BatteryMode
	}{
		{api.BatteryUnknown, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryUnknown, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryUnknown, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryNormal, api.BatteryUnknown, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryNormal, api.BatteryUnknown},
		{api.BatteryNormal, api.BatteryCharge, api.BatteryCharge},

		{api.BatteryHold, api.BatteryUnknown, api.BatteryNormal}, // return to normal
		{api.BatteryHold, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryHold, api.BatteryHold, api.BatteryUnknown},

		{api.BatteryCharge, api.BatteryUnknown, api.BatteryNormal}, // return to normal
		{api.BatteryCharge, api.BatteryNormal, api.BatteryNormal},
		{api.BatteryCharge, api.BatteryCharge, api.BatteryUnknown},
	} {
		t.Logf("%+v", tc)

		ctrl := gomock.NewController(t)

		var bat api.Meter
		batCon := api.NewMockBatteryController(ctrl)

		bat = &struct {
			api.Meter
			api.BatteryController
		}{
			BatteryController: batCon,
		}

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
			batteryMode:   tc.internal,
		}

		// 1. set required external mode
		site.SetBatteryModeExternal(tc.external)
		assert.Equal(t, site.batteryModeExternal, tc.external, "external mode expected %s got %s", tc.external, site.batteryModeExternal)
		assert.Equal(t, site.batteryMode, tc.internal, "internal mode expected unchanged %s got %s", tc.internal, site.batteryMode)

		// 2. verify external mode applied to battery
		if tc.expected != api.BatteryUnknown {
			batCon.EXPECT().SetBatteryMode(tc.expected).Times(1)
		}
		site.updateBatteryMode(false, api.Rate{})
		if !ctrl.Satisfied() {
			ctrl.Finish()
		}

		// 3. verify required external mode only applied once
		site.updateBatteryMode(false, api.Rate{})
		if !ctrl.Satisfied() {
			ctrl.Finish()
		}

		// 4. verify timer expiry
		site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
		site.batteryModeWatchdogExpired()

		// mode reverted to unknown, timer still active
		assert.Equal(t, site.batteryModeExternal, api.BatteryUnknown)
		assert.False(t, site.batteryModeExternalTimer.IsZero())

		// battery switched back to normal mode
		batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
		site.updateBatteryMode(false, api.Rate{})

		// timer disabled
		assert.True(t, site.batteryModeExternalTimer.IsZero())

		ctrl.Finish()
	}
}

func TestForcedBatteryChargeLimits(t *testing.T) {
	limit := 80.0

	for _, tc := range []struct {
		internal, expected api.BatteryMode
		soc                float64
	}{
		{api.BatteryUnknown, api.BatteryCharge, 50},
		{api.BatteryUnknown, api.BatteryHold, 90},

		{api.BatteryNormal, api.BatteryCharge, 50},
		{api.BatteryNormal, api.BatteryHold, 90},

		{api.BatteryHold, api.BatteryCharge, 50},
		{api.BatteryHold, api.BatteryHold, 90}, // TODO make this api.BatteryUnknown

		{api.BatteryCharge, api.BatteryUnknown, 50},
		{api.BatteryCharge, api.BatteryHold, 90},
	} {
		t.Logf("%+v", tc)

		ctrl := gomock.NewController(t)

		var bat api.Meter
		batSoc := api.NewMockBattery(ctrl)
		batCon := api.NewMockBatteryController(ctrl)
		batSocLimit := api.NewMockBatterySocLimiter(ctrl)

		bat = &struct {
			api.Meter
			api.Battery
			api.BatteryController
			api.BatterySocLimiter
		}{
			Meter:             bat,
			Battery:           batSoc,
			BatteryController: batCon,
			BatterySocLimiter: batSocLimit,
		}

		site := &Site{
			log:           util.NewLogger("foo"),
			batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
			batteryMode:   tc.internal,
		}

		batSoc.EXPECT().Soc().Return(tc.soc, nil).Times(1)
		batSocLimit.EXPECT().GetSocLimits().Return(0.0, limit).Times(1)

		if tc.expected != api.BatteryUnknown {
			batCon.EXPECT().SetBatteryMode(tc.expected).Times(1)
		}

		site.updateBatteryMode(true, api.Rate{})

		ctrl.Finish()
	}
}

func TestChargeToSocResolution(t *testing.T) {
	for _, tc := range []struct {
		name       string
		batterySoc float64
		targetSoc  float64
		expectedHW api.BatteryMode // hardware mode applied
	}{
		{"below target allows pv charging", 50, 80, api.BatteryNormal},
		{"at target stops charging", 80, 80, api.BatteryNoCharge},
		{"above target stops charging", 90, 80, api.BatteryNoCharge},
		{"zero target defaults to normal", 50, 0, api.BatteryNormal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			var bat api.Meter
			batCon := api.NewMockBatteryController(ctrl)

			bat = &struct {
				api.Meter
				api.BatteryController
			}{
				BatteryController: batCon,
			}

			site := &Site{
				log:                    util.NewLogger("foo"),
				batteryMeters:          []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
				batteryMode:            api.BatteryNormal,
				batteryModeExternal:    api.BatteryChargeToSoc,
				batteryModeExternalSoc: tc.targetSoc,
				battery:                types.BatteryState{Soc: tc.batterySoc},
			}

			batCon.EXPECT().SetBatteryMode(tc.expectedHW).Times(1)
			site.updateBatteryMode(false, api.Rate{})

			assert.Equal(t, tc.expectedHW, site.batteryMode)

			ctrl.Finish()
		})
	}
}

func TestChargeToSocTransition(t *testing.T) {
	ctrl := gomock.NewController(t)

	var bat api.Meter
	batCon := api.NewMockBatteryController(ctrl)

	bat = &struct {
		api.Meter
		api.BatteryController
	}{
		BatteryController: batCon,
	}

	site := &Site{
		log:                    util.NewLogger("foo"),
		batteryMeters:          []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		batteryMode:            api.BatteryNormal,
		batteryModeExternal:    api.BatteryChargeToSoc,
		batteryModeExternalSoc: 80,
		battery:                types.BatteryState{Soc: 50},
	}

	// 1. SOC below target → normal (pv charging only)
	batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
	site.updateBatteryMode(false, api.Rate{})
	assert.Equal(t, api.BatteryNormal, site.batteryMode)

	// 2. SOC reaches target → nocharge
	site.battery.Soc = 80
	batCon.EXPECT().SetBatteryMode(api.BatteryNoCharge).Times(1)
	site.updateBatteryMode(false, api.Rate{})
	assert.Equal(t, api.BatteryNoCharge, site.batteryMode)

	// 3. SOC drops below target → normal again
	site.battery.Soc = 75
	batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
	site.updateBatteryMode(false, api.Rate{})
	assert.Equal(t, api.BatteryNormal, site.batteryMode)

	ctrl.Finish()
}

func TestChargeToSocWatchdog(t *testing.T) {
	ctrl := gomock.NewController(t)

	var bat api.Meter
	batCon := api.NewMockBatteryController(ctrl)

	bat = &struct {
		api.Meter
		api.BatteryController
	}{
		BatteryController: batCon,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		batteryMode:   api.BatteryNormal,
		battery:       types.BatteryState{Soc: 50},
	}

	// set external ChargeToSoc mode
	site.SetBatteryModeExternalSoc(api.BatteryChargeToSoc, 80)
	assert.Equal(t, api.BatteryChargeToSoc, site.batteryModeExternal)
	assert.Equal(t, 80.0, site.batteryModeExternalSoc)

	// apply → normal (pv charging only)
	batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
	site.updateBatteryMode(false, api.Rate{})

	// expire watchdog
	site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
	site.batteryModeWatchdogExpired()

	// mode reverted, soc cleared
	assert.Equal(t, api.BatteryUnknown, site.batteryModeExternal)
	assert.Equal(t, 0.0, site.batteryModeExternalSoc)

	// battery returns to normal
	batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
	site.updateBatteryMode(false, api.Rate{})

	assert.True(t, site.batteryModeExternalTimer.IsZero())

	ctrl.Finish()
}
