package core

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestBatterySocRetainOnReadError guards that a failed soc read keeps the last
// known soc instead of reporting the pack as empty (discussion #26560).
func TestBatterySocRetainOnReadError(t *testing.T) {
	ctrl := gomock.NewController(t)

	meter := api.NewMockMeter(ctrl)
	meter.EXPECT().CurrentPower().Return(0.0, nil).AnyTimes()

	battery := api.NewMockBattery(ctrl)
	battery.EXPECT().Soc().Return(0.0, errors.New("read failed")).AnyTimes()

	var bat api.Meter = &struct {
		api.Meter
		api.Battery
	}{
		Meter:   meter,
		Battery: battery,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
	}
	site.battery.Soc = 84

	site.updateBatteryMeters()

	assert.Equal(t, 84.0, site.battery.Soc, "soc retained when the read fails")
}

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
		batCon := batteryControllerMock(ctrl)

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
		site.updateBatteryMode(false, false, api.Rate{})

		if tc.internal != api.BatteryNormal {
			assert.Equal(t, tc.expected, site.batteryMode)
		}

		ctrl.Finish()
	}
}

// battery controller supporting the modes a soc limit can implement
func batteryControllerMock(ctrl *gomock.Controller) *api.MockBatteryController {
	batCon := api.NewMockBatteryController(ctrl)
	batCon.EXPECT().BatteryModes().Return([]api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge, api.BatteryDischarge}).AnyTimes()

	return batCon
}

// battery meter with soc, controller and soc limits
func batteryControlMock(ctrl *gomock.Controller, soc, maxSoc float64) (api.Meter, *api.MockBatteryController) {
	batSoc := api.NewMockBattery(ctrl)
	batSoc.EXPECT().Soc().Return(soc, nil).AnyTimes()

	batSocLimit := api.NewMockBatterySocLimiter(ctrl)
	batSocLimit.EXPECT().GetSocLimits().Return(0.0, maxSoc).AnyTimes()

	batCon := batteryControllerMock(ctrl)

	return &struct {
		api.Meter
		api.Battery
		api.BatteryController
		api.BatterySocLimiter
	}{
		Battery:           batSoc,
		BatteryController: batCon,
		BatterySocLimiter: batSocLimit,
	}, batCon
}

// TestBatteryHoldAppliedOnce guards that reaching max soc during grid charge switches
// the battery to hold mode once instead of on every update
func TestBatteryHoldAppliedOnce(t *testing.T) {
	ctrl := gomock.NewController(t)

	bat, batCon := batteryControlMock(ctrl, 90, 80)

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{Name: "bat"}, bat)},
		batteryMode:   api.BatteryCharge,
	}

	batCon.EXPECT().SetBatteryMode(api.BatteryHold).Times(1)

	for range 3 {
		site.updateBatteryMode(true, false, api.Rate{})
	}

	ctrl.Finish()
}

// TestBatteryHoldNotShared guards that one battery reaching max soc does not put the
// remaining batteries into hold mode
func TestBatteryHoldNotShared(t *testing.T) {
	ctrl := gomock.NewController(t)

	full, fullCon := batteryControlMock(ctrl, 90, 80)
	empty, emptyCon := batteryControlMock(ctrl, 50, 80)

	site := &Site{
		log: util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{
			config.NewStaticDevice(config.Named{Name: "full"}, full),
			config.NewStaticDevice(config.Named{Name: "empty"}, empty),
		},
		batteryMode: api.BatteryCharge,
	}

	fullCon.EXPECT().SetBatteryMode(api.BatteryHold).Times(1)
	emptyCon.EXPECT().SetBatteryMode(gomock.Any()).Times(0)

	site.updateBatteryMode(true, false, api.Rate{})

	ctrl.Finish()
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

		mode := site.requiredBatteryMode(false, false, api.Rate{})
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
		batCon := batteryControllerMock(ctrl)

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
		site.updateBatteryMode(false, false, api.Rate{})
		if !ctrl.Satisfied() {
			ctrl.Finish()
		}

		// 3. verify required external mode only applied once
		site.updateBatteryMode(false, false, api.Rate{})
		if !ctrl.Satisfied() {
			ctrl.Finish()
		}

		// 4. verify timer expiry
		site.batteryModeExternalTimer = site.batteryModeExternalTimer.Add(-time.Hour)
		site.batteryModeWatchdogExpired()

		// mode reverted to unknown, timer still active
		assert.Equal(t, site.batteryModeExternal, api.BatteryUnknown)
		assert.False(t, site.batteryModeExternalTimer.IsZero())

		// battery switched back to normal mode unless already applied in step 2
		if tc.expected != api.BatteryNormal {
			batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)
		}
		site.updateBatteryMode(false, false, api.Rate{})

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
		batCon := batteryControllerMock(ctrl)
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

		site.updateBatteryMode(true, false, api.Rate{})

		ctrl.Finish()
	}
}

func TestForcedBatteryDischargeLimits(t *testing.T) {
	reserve := 20.0

	for _, tc := range []struct {
		internal, expected api.BatteryMode
		soc                float64
	}{
		{api.BatteryUnknown, api.BatteryDischarge, 50},
		{api.BatteryUnknown, api.BatteryHold, 10},

		{api.BatteryNormal, api.BatteryDischarge, 50},
		{api.BatteryNormal, api.BatteryHold, 10},

		{api.BatteryHold, api.BatteryDischarge, 50},
		{api.BatteryHold, api.BatteryHold, 10}, // TODO make this api.BatteryUnknown

		{api.BatteryDischarge, api.BatteryUnknown, 50},
		// steady-state re-validation: mode already Discharge and unchanged, reserve
		// reached mid-cycle; applyBatteryMode must still run to catch the reserve
		{api.BatteryDischarge, api.BatteryHold, 10},
	} {
		t.Logf("%+v", tc)

		ctrl := gomock.NewController(t)

		var bat api.Meter
		batSoc := api.NewMockBattery(ctrl)
		batCon := batteryControllerMock(ctrl)
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
		batSocLimit.EXPECT().GetSocLimits().Return(reserve, 0.0).Times(1)

		if tc.expected != api.BatteryUnknown {
			batCon.EXPECT().SetBatteryMode(tc.expected).Times(1)
		}

		site.updateBatteryMode(false, true, api.Rate{})

		ctrl.Finish()
	}
}

// TestBatteryDischargeHemsCurtailed ensures grid discharge stops but the battery
// keeps serving house load while the grid operator curtails production.
func TestBatteryDischargeHemsCurtailed(t *testing.T) {
	ctrl := gomock.NewController(t)

	var bat api.Meter
	batCon := batteryControllerMock(ctrl)

	bat = &struct {
		api.Meter
		api.BatteryController
	}{
		Meter:             bat,
		BatteryController: batCon,
	}

	curtailed := 60
	hems := api.NewMockHEMS(ctrl)
	hems.EXPECT().CurtailedPercent().Return(&curtailed).AnyTimes()
	hems.EXPECT().MaxConsumptionPower().Return(nil).AnyTimes()

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
		hems:          hems,
	}

	batCon.EXPECT().SetBatteryMode(api.BatteryNormal).Times(1)

	site.updateBatteryMode(false, true, api.Rate{})

	ctrl.Finish()
}

// TestBatteryGridDischargeEvFastCharging ensures grid discharge is held back while an EV
// is fast charging, regardless of the (opt-in, off by default) batteryDischargeControl
// toggle - forcing the battery to sell while an EV needs a fast charge is a materially
// worse outcome than the toggle's original, softer self-consumption case.
func TestBatteryGridDischargeEvFastCharging(t *testing.T) {
	lp := &Loadpoint{status: api.StatusC, mode: api.ModeNow}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{nil},
		loadpoints:    []*Loadpoint{lp},
	}

	res := site.requiredBatteryMode(false, true, api.Rate{})
	assert.Equal(t, api.BatteryHold, res, "expected discharge to be held back for a fast charging EV")
}

// TestBatteryGridDischargeActive ensures the limit only acts once the experimental
// grid discharge setting is enabled.
func TestBatteryGridDischargeActive(t *testing.T) {
	limit := 0.2
	rate := api.Rate{Value: 0.3}

	site := &Site{
		log:                       util.NewLogger("foo"),
		batteryGridDischargeLimit: &limit,
	}

	assert.False(t, site.batteryGridDischargeActive(rate), "expected limit to be inert while grid discharge is disabled")

	site.batteryGridDischarge = true
	assert.True(t, site.batteryGridDischargeActive(rate), "expected limit to act once grid discharge is enabled")
}

// TestBatteryGridDischargeLimitRequiresOptIn ensures a limit cannot outlive the
// experimental grid discharge setting: it is refused while the setting is off and
// dropped when the setting is turned off.
func TestBatteryGridDischargeLimitRequiresOptIn(t *testing.T) {
	ctrl := gomock.NewController(t)

	var bat api.Meter = &struct {
		api.Meter
		api.BatteryController
	}{
		BatteryController: api.NewMockBatteryController(ctrl),
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, bat)},
	}

	limit := 0.2

	assert.ErrorIs(t, site.SetBatteryGridDischargeLimit(&limit), ErrBatteryGridDischargeNotAvailable)
	assert.Nil(t, site.GetBatteryGridDischargeLimit(), "expected limit to be refused while grid discharge is disabled")

	assert.NoError(t, site.SetBatteryGridDischarge(true))
	assert.NoError(t, site.SetBatteryGridDischargeLimit(&limit))
	assert.Equal(t, &limit, site.GetBatteryGridDischargeLimit())

	assert.NoError(t, site.SetBatteryGridDischarge(false))
	assert.Nil(t, site.GetBatteryGridDischargeLimit(), "expected limit to be dropped with grid discharge")
}

// TestEvFastChargingActiveDisabledLoadpoint guards against the nil entries
// Loadpoints() returns for disabled loadpoints
func TestEvFastChargingActiveDisabledLoadpoint(t *testing.T) {
	site := &Site{loadpoints: []*Loadpoint{nil}}

	assert.False(t, site.evFastChargingActive())
}
