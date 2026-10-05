package core

import (
	"fmt"
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

	for _, k := range []string{keys.Experimental, keys.Optimizer} {
		settings.SetBool(k, true)
	}
	settings.SetString(keys.OptimizerAutomatic, OptimizerAutomaticFull)

	t.Cleanup(func() {
		sponsor.Subject = subject
		for _, k := range []string{keys.Experimental, keys.Optimizer} {
			settings.SetBool(k, false)
		}
		settings.SetString(keys.OptimizerAutomatic, OptimizerAutomaticOff)
	})
}

// TestAutomaticLevels verifies that the battery level controls the battery only
func TestAutomaticLevels(t *testing.T) {
	enableAutomatic(t)
	site := &Site{}

	for _, tc := range []struct {
		stored              string
		battery, loadpoints bool
	}{
		{OptimizerAutomaticOff, false, false},
		{OptimizerAutomaticBattery, true, false},
		{OptimizerAutomaticFull, true, true},
		{"true", true, true}, // boolean switch before the levels
		{"false", false, false},
	} {
		settings.SetString(keys.OptimizerAutomatic, tc.stored)
		assert.Equal(t, tc.battery, site.Automatic(), tc.stored)
		assert.Equal(t, tc.loadpoints, site.AutomaticLoadpoints(), tc.stored)
	}
}

func automaticLoadpoint(t *testing.T, ac api.AlwaysCharge, automatic bool) (*Loadpoint, *api.MockCharger, *gomock.Controller) {
	t.Helper()

	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	Voltage = 230

	lp := &Loadpoint{
		log:          util.NewLogger("foo"),
		bus:          evbus.New(),
		clock:        clock.NewMock(),
		charger:      charger,
		chargeMeter:  newChargeMeter(&Null{}),
		chargeRater:  &Null{},
		chargeTimer:  &Null{},
		wakeUpTimer:  NewTimer(),
		minCurrent:   minA,
		maxCurrent:   maxA,
		phases:       1,
		status:       api.StatusC,
		mode:         api.ModeSmart,
		alwaysCharge: ac,
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

	// the 1p test loadpoint tops out at 3680W
	full := types.Suggestion{Action: actionCharge, Charge: 3680, Grid: 1000}
	stop := types.Suggestion{Action: actionStop}

	tc := []struct {
		mode   api.ChargeMode
		ac     api.AlwaysCharge
		s      types.Suggestion
		expect func(h *api.MockCharger)
	}{
		// optimizer starts and stops smart charging, replacing the price limits.
		// Stop hands over to the pv loop, which disables at once after a grid-fed slot
		{api.ModeSmart, api.AlwaysChargeOff, full, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(maxA)) }},
		{api.ModeSmart, api.AlwaysChargeOff, stop, func(h *api.MockCharger) { h.EXPECT().Enable(false) }},

		// always charge keeps its minimum power when the optimizer stops
		{api.ModeSmart, api.AlwaysChargeOn, full, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(maxA)) }},
		{api.ModeSmart, api.AlwaysChargeOn, stop, nil}, // already at min current

		// a grid-fed power below the maximum is applied as current
		{api.ModeSmart, api.AlwaysChargeOff, types.Suggestion{Action: actionCharge, Charge: 2300, Grid: 1000}, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(10)) }},

		// off and fast remain the user's decision
		{api.ModeOff, api.AlwaysChargeOff, full, func(h *api.MockCharger) { h.EXPECT().Enable(false) }},
		{api.ModeNow, api.AlwaysChargeOff, stop, func(h *api.MockCharger) { h.EXPECT().MaxCurrent(int64(maxA)) }},
	}

	for _, tc := range tc {
		t.Log(tc)

		lp, charger, ctrl := automaticLoadpoint(t, tc.ac, true)
		lp.mode = tc.mode
		lp.clock.(*clock.Mock).Add(time.Hour) // elapsed must lie in the past
		lp.pvTimer = elapsed                  // previous grid-fed slot
		lp.setSuggestion(&tc.s)

		if tc.expect != nil {
			tc.expect(charger)
		}

		// grid import above min power, no measured surplus
		lp.Update(2000, 0, nil, nil, false, false, 0, nil, nil, nil)

		ctrl.Finish()
	}
}

// TestOptimizerSurplusRegime covers a charge power matched to the forecast surplus:
// control falls through to the pv loop and its enable/disable timer keeps running
// instead of being elapsed on every cycle
func TestOptimizerSurplusRegime(t *testing.T) {
	enableAutomatic(t)

	lp, charger, ctrl := automaticLoadpoint(t, api.AlwaysChargeOff, true)
	lp.Disable.Delay = time.Minute
	lp.offeredCurrent = maxA

	// a previous grid-fed slot left the pv timer elapsed
	lp.pvTimer = elapsed

	lp.setSuggestion(&types.Suggestion{Action: actionCharge, Charge: 2300})

	// surplus dip: the loadpoint steps down to min current while the disable
	// timer runs instead of disabling right away
	charger.EXPECT().MaxCurrent(int64(minA))
	lp.Update(4000, 0, nil, nil, false, false, 0, nil, nil, nil)

	assert.False(t, lp.pvTimer.Equal(elapsed), "pv timer must not stay elapsed")
	assert.False(t, lp.pvTimer.IsZero(), "pv disable timer must be running")

	ctrl.Finish()
}

// TestOptimizerStopFollowsSurplus covers a stop based on a forecast surplus that is
// too low: the pv loop still enables on measured surplus after the enable delay
func TestOptimizerStopFollowsSurplus(t *testing.T) {
	enableAutomatic(t)

	stop := &types.Suggestion{Action: actionStop}

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.site = &mockSite{automatic: true}
	lp.vehicle = modelledVehicle(gomock.NewController(t))

	// disabled: a timer elapsed by a grid-fed slot must not skip the enable delay
	lp.pvTimer = elapsed
	handled, err := lp.optimizerCharging(stop)
	assert.NoError(t, err)
	assert.False(t, handled, "stop must leave the decision to the pv loop")
	assert.True(t, lp.pvTimer.IsZero(), "enable delay must apply")

	// enabled after a grid-fed slot: disable without measured surplus at once
	lp.enabled = true
	lp.pvTimer = elapsed
	handled, _ = lp.optimizerCharging(stop)
	assert.False(t, handled)
	assert.True(t, lp.pvTimer.Equal(elapsed))
}

// TestOptimizerFlexibility covers a loadpoint the optimizer pins to a setpoint:
// it does not yield to a higher priority loadpoint, so its power is not flexible
func TestOptimizerFlexibility(t *testing.T) {
	enableAutomatic(t)
	Voltage = 230

	for _, tc := range []struct {
		s    types.Suggestion
		want float64
	}{
		{types.Suggestion{Action: actionCharge, Charge: 3680, Grid: 1000}, 0}, // full power
		{types.Suggestion{Action: actionCharge, Charge: 2300, Grid: 1000}, 0}, // grid-fed setpoint
		{types.Suggestion{Action: actionCharge, Charge: 2300}, 2700},          // surplus regime, pv loop yields
		{types.Suggestion{Action: actionStop}, 2700},                          // stop, pv loop yields
	} {
		lp := NewLoadpoint(util.NewLogger("foo"), nil)
		lp.mode = api.ModeSmart
		lp.status = api.StatusC
		lp.chargePower = 2700
		lp.phases = 1
		lp.vehicle = modelledVehicle(gomock.NewController(t))
		lp.site = &mockSite{automatic: true}
		lp.setSuggestion(&tc.s)

		assert.Equal(t, tc.want, lp.GetChargePowerFlexibility(nil), tc.s)
	}
}

func TestOptimizerGateInactive(t *testing.T) {
	enableAutomatic(t)

	// automatic disabled: pv surplus decides, the suggestion is advisory only
	lp, _, ctrl := automaticLoadpoint(t, api.AlwaysChargeOff, false)
	lp.setSuggestion(&types.Suggestion{Action: actionCharge})

	assert.Nil(t, lp.gate())

	// pv is balanced at zero site power, a gated loadpoint would go to max current
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)

	ctrl.Finish()
}

func TestOptimizerGateStale(t *testing.T) {
	enableAutomatic(t)

	// a stalled optimizer must not keep the loadpoint gated
	lp, _, ctrl := automaticLoadpoint(t, api.AlwaysChargeOff, true)
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

func TestBatteryModeAutomatic(t *testing.T) {
	enableAutomatic(t)

	ctrl := gomock.NewController(t)
	batCon := batteryControllerMock(ctrl)

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

	// an expired solve releases the battery
	site.clearSuggestions()

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

// TestOptimizerPhaseScaleUp covers a grid-fed charge on 1p behind a circuit:
// the scale up delay must run across control cycles instead of restarting on each
func TestOptimizerPhaseScaleUp(t *testing.T) {
	for _, charge := range []float64{11040, 9200} { // full power, and more than 1p delivers
		t.Run(fmt.Sprintf("%.0fW", charge), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			clck := clock.NewMock()
			clck.Add(time.Hour)

			Voltage = 230

			lp := NewLoadpoint(util.NewLogger("foo"), nil)
			lp.clock = clck
			lp.minCurrent = minA
			lp.maxCurrent = maxA
			lp.Enable.Delay = 3 * time.Minute
			lp.phases = 1
			lp.status = api.StatusC
			lp.chargePower = 3680
			lp.wakeUpTimer = NewTimer()

			plainCharger := api.NewMockCharger(ctrl)
			phaseCharger := api.NewMockPhaseSwitcher(ctrl)
			lp.charger = struct {
				*api.MockCharger
				*api.MockPhaseSwitcher
			}{plainCharger, phaseCharger}

			plainCharger.EXPECT().Enabled().Return(true, nil).AnyTimes()
			plainCharger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()
			plainCharger.EXPECT().MaxCurrent(gomock.Any()).Return(nil).AnyTimes()

			circuit := api.NewMockCircuit(ctrl)
			lp.circuit = circuit
			circuit.EXPECT().GetMaxPower().Return(0.0).AnyTimes()
			circuit.EXPECT().ValidatePower(gomock.Any(), gomock.Any()).DoAndReturn(func(_, new float64) float64 { return new }).AnyTimes()
			circuit.EXPECT().ValidateCurrent(gomock.Any(), gomock.Any()).DoAndReturn(func(_, new float64) float64 { return new }).AnyTimes()

			s := &types.Suggestion{Action: actionCharge, Charge: charge, Grid: charge}

			handled, err := lp.optimizerCharging(s)
			assert.True(t, handled)
			assert.NoError(t, err)
			started := lp.phaseTimer
			assert.False(t, started.IsZero(), "scale up timer must be running")

			// next cycle keeps the timer
			clck.Add(time.Minute)
			_, err = lp.optimizerCharging(s)
			assert.NoError(t, err)
			assert.Equal(t, started, lp.phaseTimer, "scale up timer must not restart")

			// delay elapsed
			clck.Add(lp.Enable.Delay)
			phaseCharger.EXPECT().Phases1p3p(3).Return(nil)
			_, err = lp.optimizerCharging(s)
			assert.NoError(t, err)
			assert.Equal(t, 3, lp.GetPhases())
		})
	}
}
