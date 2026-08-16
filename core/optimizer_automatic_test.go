package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	coresettings "github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// enableAutomatic puts the optimizer in control for the duration of the test
func enableAutomatic(t *testing.T) {
	t.Helper()

	subject := sponsor.Subject
	sponsor.Subject = "test"

	for _, k := range []string{keys.Experimental, keys.Optimizer, keys.OptimizerAutomatic} {
		settings.SetBool(k, true)
	}

	t.Cleanup(func() {
		sponsor.Subject = subject
		for _, k := range []string{keys.Experimental, keys.Optimizer, keys.OptimizerAutomatic} {
			settings.SetBool(k, false)
		}
	})
}

func automaticLoadpoint(t *testing.T, mode api.ChargeMode, automatic bool) (*Loadpoint, *api.MockCharger, *gomock.Controller) {
	t.Helper()

	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock.NewMock(),
		charger:     charger,
		chargeMeter: &Null{},
		chargeRater: &Null{},
		chargeTimer: &Null{},
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		phases:      1,
		status:      api.StatusC,
		mode:        mode,
	}

	// only vehicles with known capacity can be modelled by the optimizer
	lp.vehicle = modelledVehicle(ctrl)

	attachListeners(t, lp)

	// attachListeners assigns a real site
	lp.site = &mockSite{automatic: automatic}

	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().Enabled().Return(true, nil)

	return lp, charger, ctrl
}

// modelledVehicle returns a vehicle the optimizer can model as storage
func modelledVehicle(ctrl *gomock.Controller) api.Vehicle {
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().Capacity().Return(50.0).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().GetTitle().Return("").AnyTimes()
	v.EXPECT().Icon().Return("").AnyTimes()
	v.EXPECT().Identifiers().Return(nil).AnyTimes()
	v.EXPECT().Soc().Return(50.0, nil).AnyTimes()
	return v
}

func TestOptimizerGate(t *testing.T) {
	enableAutomatic(t)

	tc := []struct {
		mode   api.ChargeMode
		action string
		expect func(h *api.MockCharger)
	}{
		// optimizer starts and stops pv charging, replacing the price limits
		{api.ModePV, actionCharge, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(maxA)) }},
		{api.ModePV, actionStop, func(h *api.MockCharger) { h.EXPECT().Enable(false) }},

		// minpv keeps its minimum power when the optimizer stops
		{api.ModeMinPV, actionCharge, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(maxA)) }},
		{api.ModeMinPV, actionStop, nil}, // already at min current

		// off and fast remain the user's decision
		{api.ModeOff, actionCharge, func(h *api.MockCharger) { h.EXPECT().Enable(false) }},
		{api.ModeNow, actionStop, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(maxA)) }},
	}

	for _, tc := range tc {
		t.Log(tc)

		lp, charger, ctrl := automaticLoadpoint(t, tc.mode, true)
		lp.setSuggestion(&types.Suggestion{Action: tc.action})

		if tc.expect != nil {
			tc.expect(charger)
		}

		lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)

		ctrl.Finish()
	}
}

func TestOptimizerGateInactive(t *testing.T) {
	enableAutomatic(t)

	// automatic disabled: pv surplus decides, the suggestion is advisory only
	lp, _, ctrl := automaticLoadpoint(t, api.ModePV, false)
	lp.setSuggestion(&types.Suggestion{Action: actionCharge})

	assert.Nil(t, lp.gate())

	// pv is balanced at zero site power, a gated loadpoint would go to max current
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)

	ctrl.Finish()
}

func TestOptimizerGateStale(t *testing.T) {
	enableAutomatic(t)

	// a stalled optimizer must not keep the loadpoint gated
	lp, _, ctrl := automaticLoadpoint(t, api.ModePV, true)
	lp.setSuggestion(&types.Suggestion{Action: actionCharge})
	lp.suggestionUpdated = lp.clock.Now().Add(-suggestionMaxAge - time.Minute)

	assert.Nil(t, lp.gate())

	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)

	ctrl.Finish()
}

func TestSmartCostLimitUnavailable(t *testing.T) {
	enableAutomatic(t)

	limit := 0.2

	lp := &Loadpoint{
		log:      util.NewLogger("foo"),
		clock:    clock.NewMock(),
		settings: coresettings.NewDatabaseSettingsAdapter("test"),
	}
	lp.site = &mockSite{automatic: true}
	lp.vehicle = modelledVehicle(gomock.NewController(t))

	assert.ErrorIs(t, lp.SetSmartCostLimit(&limit), ErrOptimizerAutomatic)
	assert.ErrorIs(t, lp.SetSmartFeedInPriorityLimit(&limit), ErrOptimizerAutomatic)
	assert.Nil(t, lp.GetSmartCostLimit())

	// clearing is a no-op, so a config round-trip does not discard the stored limit
	assert.NoError(t, lp.SetSmartCostLimit(nil))

	// loadpoints the optimizer cannot model keep their limits
	lp.charger = struct {
		api.Charger
		api.FeatureDescriber
	}{FeatureDescriber: &featureCharger{features: []api.Feature{api.Heating}}}

	assert.NoError(t, lp.SetSmartCostLimit(&limit))
	assert.Equal(t, &limit, lp.GetSmartCostLimit())

	// a vehicle without known capacity cannot be modelled either
	lp.charger = nil
	lp.vehicle = nil

	assert.NoError(t, lp.SetSmartFeedInPriorityLimit(&limit))
	assert.Equal(t, &limit, lp.GetSmartFeedInPriorityLimit())
}

type featureCharger struct {
	features []api.Feature
}

func (c *featureCharger) Features() []api.Feature {
	return c.features
}

func TestBatteryModeAutomatic(t *testing.T) {
	enableAutomatic(t)

	ctrl := gomock.NewController(t)
	batCon := api.NewMockBatteryController(ctrl)

	var bat api.Meter = &struct {
		api.Meter
		api.BatteryController
	}{
		BatteryController: batCon,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{Name: "bat"}, bat)},
	}

	// optimizer decides to grid charge, replacing the grid charge limit
	site.setSuggestions(map[string]types.Suggestion{
		batteryKey("bat"): {Action: api.BatteryCharge.String()},
	})

	batCon.EXPECT().SetBatteryMode(api.BatteryCharge)
	site.updateBatteryMode(false, false, api.Rate{})
	assert.Equal(t, api.BatteryCharge, site.GetBatteryMode())

	// a stalled optimizer releases the battery
	site.suggestionsUpdated = time.Now().Add(-suggestionMaxAge - time.Minute)

	batCon.EXPECT().SetBatteryMode(api.BatteryNormal)
	site.updateBatteryMode(false, false, api.Rate{})
	assert.Equal(t, api.BatteryNormal, site.GetBatteryMode())

	ctrl.Finish()
}

func TestBatteryGridChargeLimitUnavailable(t *testing.T) {
	enableAutomatic(t)

	ctrl := gomock.NewController(t)
	batCon := api.NewMockBatteryController(ctrl)

	var bat api.Meter = &struct {
		api.Meter
		api.BatteryController
	}{
		BatteryController: batCon,
	}

	site := &Site{
		log:           util.NewLogger("foo"),
		batteryMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{Name: "bat"}, bat)},
	}

	limit := 0.2
	assert.ErrorIs(t, site.SetBatteryGridChargeLimit(&limit), ErrOptimizerAutomatic)
	assert.ErrorIs(t, site.SetBatteryDischargeControl(true), ErrOptimizerAutomatic)
}
