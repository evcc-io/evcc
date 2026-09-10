package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/messenger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	minA float64 = 6
	maxA float64 = 16
)

type Null struct{}

func (n *Null) CurrentPower() (float64, error) {
	return 0, nil
}

func (n *Null) ChargedEnergy() (float64, error) {
	return 0, nil
}

func (n *Null) ChargeDuration() (time.Duration, error) {
	return 0, nil
}

func createChannels(t *testing.T) (chan util.Param, chan messenger.Event, chan *Loadpoint) {
	t.Helper()

	uiChan := make(chan util.Param)
	pushChan := make(chan messenger.Event)
	lpChan := make(chan *Loadpoint)

	log := false
	go func() {
		for {
			select {
			case v := <-uiChan:
				if log {
					t.Log(v)
				}
			case v := <-pushChan:
				if log {
					t.Log(v)
				}
			case v := <-lpChan:
				if log {
					t.Log(v)
				}
			}
		}
	}()

	return uiChan, pushChan, lpChan
}

func attachChannels(lp *Loadpoint, uiChan chan util.Param, pushChan chan messenger.Event, lpChan chan *Loadpoint) {
	lp.uiChan = uiChan
	lp.pushChan = pushChan
	lp.lpChan = lpChan
}

func attachListeners(t *testing.T, lp *Loadpoint) {
	t.Helper()

	Voltage = 230 // V

	if charger, ok := lp.charger.(*api.MockCharger); ok && charger != nil {
		charger.EXPECT().Enabled().Return(true, nil)
		charger.EXPECT().MaxCurrent(int64(lp.minCurrent)).Return(nil)
	}

	uiChan, pushChan, lpChan := createChannels(t)
	lp.Prepare(new(Site), uiChan, pushChan, lpChan)
}

func TestNew(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)

	if lp.phases != 0 {
		t.Errorf("Phases %v", lp.phases)
	}
	if lp.maxCurrent != maxA {
		t.Errorf("MaxCurrent %v", lp.maxCurrent)
	}
	if lp.status != api.StatusNone {
		t.Errorf("status %v", lp.status)
	}
	if lp.charging() {
		t.Errorf("charging %v", lp.charging())
	}
}

func TestUpdatePowerZero(t *testing.T) {
	tc := []struct {
		status       api.ChargeStatus
		mode         api.ChargeMode
		alwaysCharge api.AlwaysCharge
		expect       func(h *api.MockCharger)
	}{
		{api.StatusA, api.ModeOff, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusA, api.ModeNow, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusA, api.ModeSmart, api.AlwaysChargeOn, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusA, api.ModeSmart, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false) // zero since update called with 0
		}},

		{api.StatusB, api.ModeOff, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusB, api.ModeNow, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().MaxCurrent(int64(maxA)) // true
		}},
		{api.StatusB, api.ModeSmart, api.AlwaysChargeOn, func(h *api.MockCharger) {
			// MaxCurrent omitted since identical value
		}},
		{api.StatusB, api.ModeSmart, api.AlwaysChargeOff, func(h *api.MockCharger) {
			// zero since update called with 0
			// force = false due to smart mode climater check
			h.EXPECT().Enable(false)
		}},

		{api.StatusC, api.ModeOff, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusC, api.ModeNow, api.AlwaysChargeOff, func(h *api.MockCharger) {
			h.EXPECT().MaxCurrent(int64(maxA)) // true
		}},
		{api.StatusC, api.ModeSmart, api.AlwaysChargeOn, func(h *api.MockCharger) {
			// MaxCurrent omitted since identical value
		}},
		{api.StatusC, api.ModeSmart, api.AlwaysChargeOff, func(h *api.MockCharger) {
			// omitted since PV balanced
		}},
	}

	for _, tc := range tc {
		t.Log(tc)

		clock := clock.NewMock()
		ctrl := gomock.NewController(t)
		charger := api.NewMockCharger(ctrl)

		lp := &Loadpoint{
			log:         util.NewLogger("foo"),
			bus:         evbus.New(),
			clock:       clock,
			charger:     charger,
			chargeMeter: &Null{}, // silence nil panics
			chargeRater: &Null{}, // silence nil panics
			chargeTimer: &Null{}, // silence nil panics
			wakeUpTimer: NewTimer(),
			minCurrent:  minA,
			maxCurrent:  maxA,
			phases:      1,
			solarShare:  1,
			status:      tc.status, // no status change
		}

		attachListeners(t, lp)

		// initial status
		charger.EXPECT().Status().Return(tc.status, nil)
		charger.EXPECT().Enabled().Return(true, nil)

		if tc.expect != nil {
			tc.expect(charger)
		}

		lp.mode = tc.mode
		lp.alwaysCharge = tc.alwaysCharge
		lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil) // false,sitePower false,0

		ctrl.Finish()
	}
}

func TestPVHysteresis(t *testing.T) {
	const dt = time.Minute
	const phases = 3
	type se struct {
		site    float64
		delay   time.Duration // test case delay since start
		current float64
	}
	tc := []struct {
		enabled         bool
		enable, disable float64
		series          []se
	}{
		// keep disabled
		{false, 0, 0, []se{
			{0, 0, 0},
			{0, 1, 0},
			{0, dt - 1, 0},
			{0, dt + 1, 0},
		}},
		// enable when threshold not configured but min power met
		{false, 0, 0, []se{
			{-6 * 100 * phases, 0, 0},
			{-6 * 100 * phases, 1, 0},
			{-6 * 100 * phases, dt - 1, 0},
			{-6 * 100 * phases, dt + 1, minA},
		}},
		// keep disabled when threshold not configured
		{false, 0, 0, []se{
			{-400, 0, 0},
			{-400, 1, 0},
			{-400, dt - 1, 0},
			{-400, dt + 1, 0},
		}},
		// keep disabled when threshold (lower minCurrent) not met
		{false, -500, 0, []se{
			{-400, 0, 0},
			{-400, 1, 0},
			{-400, dt - 1, 0},
			{-400, dt + 1, 0},
		}},
		// keep disabled when threshold (higher minCurrent) not met
		{false, -7 * 100 * phases, 0, []se{
			{-6 * 100 * phases, 0, 0},
			{-6 * 100 * phases, 1, 0},
			{-6 * 100 * phases, dt - 1, 0},
			{-6 * 100 * phases, dt + 1, 0},
		}},
		// enable when threshold met
		{false, -500, 0, []se{
			{-500, 0, 0},
			{-500, 1, 0},
			{-500, dt - 1, 0},
			{-500, dt + 1, minA},
		}},
		// keep enabled at max
		{true, 500, 0, []se{
			{-16 * 100 * phases, 0, maxA},
			{-16 * 100 * phases, 1, maxA},
			{-16 * 100 * phases, dt - 1, maxA},
			{-16 * 100 * phases, dt + 1, maxA},
		}},
		// keep enabled at min
		{true, 500, 0, []se{
			{-6 * 100 * phases, 0, minA},
			{-6 * 100 * phases, 1, minA},
			{-6 * 100 * phases, dt - 1, minA},
			{-6 * 100 * phases, dt + 1, minA},
		}},
		// keep enabled at min (negative threshold)
		{true, 0, 500, []se{
			{-500, 0, minA},
			{-500, 1, minA},
			{-500, dt - 1, minA},
			{-500, dt + 1, minA},
		}},
		// disable when threshold met
		{true, 0, 500, []se{
			{500, 0, minA},
			{500, 1, minA},
			{500, dt - 1, minA},
			{500, dt + 1, 0},
		}},
		// reset enable timer when threshold not met while timer active
		{false, -500, 0, []se{
			{-500, 0, 0},
			{-500, 1, 0},
			{-499, dt - 1, 0}, // should reset timer
			{-500, dt + 1, 0}, // new begin of timer
			{-500, 2 * dt, 0},
			{-500, 2*dt + 1, minA},
		}},
		// reset enable timer when threshold not met while timer active and threshold not configured
		{false, 0, 0, []se{
			{-6*100*10 - 1, dt + 1, 0},
			{-6 * 100 * phases, dt + 1, 0},
			{-6 * 100 * phases, dt + 2, 0},
			{-6 * 100 * phases, 2 * dt, 0},
			{-6 * 100 * phases, 2*dt + 1, minA},
		}},
		// reset disable timer when threshold not met while timer active
		{true, 0, 500, []se{
			{500, 0, minA},
			{500, 1, minA},
			{499, dt - 1, minA}, // reset timer
			{500, dt + 1, minA}, // within reset timer duration
			{500, 2 * dt, minA}, // still within reset timer duration
			{500, 2*dt + 1, 0},  // reset timer elapsed
		}},
	}

	for _, status := range []api.ChargeStatus{api.StatusB, api.StatusC} {
		for _, tc := range tc {
			t.Log(tc)

			clock := clock.NewMock()
			ctrl := gomock.NewController(t)
			charger := api.NewMockCharger(ctrl)

			Voltage = 100
			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				clock:          clock,
				charger:        charger,
				minCurrent:     minA,
				maxCurrent:     maxA,
				phases:         phases,
				measuredPhases: phases,
				solarShare:     1,
				Enable: loadpoint.ThresholdConfig{
					Threshold: tc.enable,
					Delay:     dt,
				},
				Disable: loadpoint.ThresholdConfig{
					Threshold: tc.disable,
					Delay:     dt,
				},
			}

			// charging, otherwise PV mode logic is short-circuited
			lp.status = status

			start := clock.Now()

			for step, se := range tc.series {
				clock.Set(start.Add(se.delay))

				// maxCurrent will read actual current and enabled state in PV mode
				// charger.EXPECT().Enabled().Return(tc.enabled, nil)

				lp.enabled = tc.enabled
				current := lp.pvMaxCurrent(se.site, 0, false, false)

				if current != se.current {
					t.Errorf("step %d: wanted %.1f, got %.1f", step, se.current, current)
				}
			}

			ctrl.Finish()
		}
	}
}

func TestPVHysteresisForStatusOtherThanC(t *testing.T) {
	const phases = 3

	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	Voltage = 100
	lp := &Loadpoint{
		log:            util.NewLogger("foo"),
		clock:          clock,
		minCurrent:     minA,
		maxCurrent:     maxA,
		phases:         phases,
		measuredPhases: phases,
		solarShare:     1,
	}

	// not connected, test PV mode logic  short-circuited
	lp.status = api.StatusA

	// maxCurrent will read enabled state in PV mode
	sitePower := -float64(phases)*minA*Voltage + 1 // 1W below min power
	current := lp.pvMaxCurrent(sitePower, 0, false, false)

	if current != 0 {
		t.Errorf("PV mode could not disable charger as expected. Expected 0, got %.f", current)
	}

	ctrl.Finish()
}

func TestDisableAndEnableAtTargetSoc(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	vehicle := api.NewMockVehicle(ctrl)

	// wrap vehicle with estimator
	expectVehiclePublish(vehicle)

	socEstimator := soc.NewEstimator(util.NewLogger("foo"), vehicle)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		charger:     charger,
		chargeMeter: &Null{},            // silence nil panics
		chargeRater: &Null{},            // silence nil panics
		chargeTimer: &Null{},            // silence nil panics
		progress:    NewProgress(0, 10), // silence nil panics
		wakeUpTimer: NewTimer(),         // silence nil panics
		// coordinator:   coordinator.NewDummy(), // silence nil panics
		minCurrent:   minA,
		maxCurrent:   maxA,
		vehicle:      vehicle,      // needed for targetSoc check
		socEstimator: socEstimator, // instead of vehicle: vehicle,
		mode:         api.ModeNow,
		limitSoc:     90, // session limit
		Soc: loadpoint.SocConfig{
			Poll: loadpoint.PollConfig{
				Mode:     loadpoint.PollConnected, // allow polling when connected
				Interval: pollInterval,
			},
		},
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.offeredCurrent = minA
	lp.status = api.StatusC

	t.Log("charging below soc target")
	vehicle.EXPECT().Soc().Return(85.0, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)
	lp.Update(500, 0, nil, nil, false, false, 0, nil, nil, nil)
	ctrl.Finish()

	t.Log("charging above target - soc deactivates charger")
	clock.Add(5 * time.Minute)
	vehicle.EXPECT().Soc().Return(90.0, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Enable(false).Return(nil)
	lp.Update(500, 0, nil, nil, false, false, 0, nil, nil, nil)
	ctrl.Finish()

	t.Log("deactivated charger changes status to B")
	clock.Add(5 * time.Minute)
	vehicle.EXPECT().Soc().Return(95.0, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	lp.Update(-500, 0, nil, nil, false, false, 0, nil, nil, nil)
	ctrl.Finish()

	t.Log("soc has risen below target - soc update prevented by timer")
	clock.Add(5 * time.Minute)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	lp.Update(-500, 0, nil, nil, false, false, 0, nil, nil, nil)
	ctrl.Finish()

	t.Log("soc has fallen below target - soc update timer expired")
	clock.Add(pollInterval)
	vehicle.EXPECT().Soc().Return(85.0, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)
	charger.EXPECT().Enable(true).Return(nil)
	lp.Update(-500, 0, nil, nil, false, false, 0, nil, nil, nil)
	ctrl.Finish()
}

func TestSetModeAndSocAtDisconnect(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		settings:    settings.NewDatabaseSettingsAdapter("foo"),
		charger:     charger,
		chargeMeter: &Null{}, // silence nil panics
		chargeRater: &Null{}, // silence nil panics
		chargeTimer: &Null{}, // silence nil panics
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		status:      api.StatusC,
		DefaultMode: api.ModeOff, // default mode
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.offeredCurrent = minA
	lp.mode = api.ModeNow

	t.Log("charging at min")
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)
	lp.Update(500, 0, nil, nil, false, false, 0, nil, nil, nil)

	t.Log("switch off when disconnected")
	clock.Add(5 * time.Minute)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusA, nil)
	charger.EXPECT().Enable(false).Return(nil)
	lp.Update(-300, 0, nil, nil, false, false, 0, nil, nil, nil)

	if mode := lp.GetMode(); mode != api.ModeOff {
		t.Error("unexpected mode", mode)
	}

	ctrl.Finish()
}

// cacheExpecter can be used to verify asynchronously written values from cache
func cacheExpecter(t *testing.T, lp *Loadpoint) (*util.ParamCache, func(key string, val any)) {
	t.Helper()

	// attach cache for verifying values
	paramC := make(chan util.Param)
	lp.uiChan = paramC

	cache := util.NewParamCache()
	go cache.Run(paramC)

	expect := func(key string, val any) {
		time.Sleep(100 * time.Millisecond) // wait for cache to catch up

		p := cache.Get(key)
		assert.Equal(t, val, p.Val, key)
	}
	return cache, expect
}

func TestChargedEnergyAtDisconnect(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	rater := api.NewMockChargeRater(ctrl)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		charger:     charger,
		chargeMeter: &Null{}, // silence nil panics
		chargeRater: rater,
		chargeTimer: &Null{}, // silence nil panics
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		status:      api.StatusC,
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.offeredCurrent = maxA
	lp.mode = api.ModeNow

	// attach cache for verifying values
	_, expectCache := cacheExpecter(t, lp)

	t.Log("start charging at 0 kWh")
	rater.EXPECT().ChargedEnergy().Return(0.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, 0, nil, nil, false, false, 0, nil, nil, nil)

	t.Log("at 1:00h charging at 5 kWh")
	clock.Add(time.Hour)
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, 0, nil, nil, false, false, 0, nil, nil, nil)
	expectCache("chargedEnergy", 5000.0)

	t.Log("at 1:00h stop charging at 5 kWh")
	clock.Add(time.Second)
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	lp.Update(-1, 0, nil, nil, false, false, 0, nil, nil, nil)
	expectCache("chargedEnergy", 5000.0)

	t.Log("at 1:00h restart charging at 5 kWh")
	clock.Add(time.Second)
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, 0, nil, nil, false, false, 0, nil, nil, nil)
	expectCache("chargedEnergy", 5000.0)

	t.Log("at 1:30h continue charging at 7.5 kWh")
	clock.Add(30 * time.Minute)
	rater.EXPECT().ChargedEnergy().Return(7.5, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, 0, nil, nil, false, false, 0, nil, nil, nil)
	expectCache("chargedEnergy", 7500.0)

	t.Log("at 2:00h stop charging at 10 kWh")
	clock.Add(30 * time.Minute)
	rater.EXPECT().ChargedEnergy().Return(10.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	lp.Update(-1, 0, nil, nil, false, false, 0, nil, nil, nil)
	expectCache("chargedEnergy", 10000.0)

	ctrl.Finish()
}

func TestSocPoll(t *testing.T) {
	clock := clock.NewMock()
	tRefresh := pollInterval
	tNoRefresh := pollInterval / 2

	lp := &Loadpoint{
		clock: clock,
		log:   util.NewLogger("foo"),
		Soc: loadpoint.SocConfig{
			Poll: loadpoint.PollConfig{
				Interval: time.Hour,
			},
		},
	}

	tc := []struct {
		mode   loadpoint.PollMode
		status api.ChargeStatus
		dt     time.Duration
		res    bool
	}{
		// pollCharging
		{loadpoint.PollCharging, api.StatusA, -1, false},
		{loadpoint.PollCharging, api.StatusA, 0, false},
		{loadpoint.PollCharging, api.StatusA, tRefresh, false},
		{loadpoint.PollCharging, api.StatusB, -1, true}, // poll once when car connected
		{loadpoint.PollCharging, api.StatusB, 0, false},
		{loadpoint.PollCharging, api.StatusB, tRefresh, false},
		{loadpoint.PollCharging, api.StatusC, -1, true},
		{loadpoint.PollCharging, api.StatusC, 0, true},
		{loadpoint.PollCharging, api.StatusC, tNoRefresh, true}, // cached by vehicle
		{loadpoint.PollCharging, api.StatusB, -1, true},         // fetch if car stopped charging
		{loadpoint.PollCharging, api.StatusB, 0, false},         // no more polling
		{loadpoint.PollCharging, api.StatusB, tRefresh, false},  // no more polling

		// pollConnected
		{loadpoint.PollConnected, api.StatusA, -1, false},
		{loadpoint.PollConnected, api.StatusA, 0, false},
		{loadpoint.PollConnected, api.StatusA, tRefresh, false},
		{loadpoint.PollConnected, api.StatusB, -1, true},
		{loadpoint.PollConnected, api.StatusB, 0, false},
		{loadpoint.PollConnected, api.StatusB, tNoRefresh, false},
		{loadpoint.PollConnected, api.StatusB, tRefresh, true},
		{loadpoint.PollConnected, api.StatusC, -1, true},
		{loadpoint.PollConnected, api.StatusC, 0, true},
		{loadpoint.PollConnected, api.StatusC, tNoRefresh, true}, // cached by vehicle
		{loadpoint.PollConnected, api.StatusC, tRefresh, true},

		// pollAlways
		{loadpoint.PollAlways, api.StatusA, -1, true},
		{loadpoint.PollAlways, api.StatusA, 0, false},
		{loadpoint.PollAlways, api.StatusA, tNoRefresh, false},
		{loadpoint.PollAlways, api.StatusA, tRefresh, true},
		{loadpoint.PollAlways, api.StatusB, -1, true},
		{loadpoint.PollAlways, api.StatusB, 0, false},
		{loadpoint.PollAlways, api.StatusB, tNoRefresh, false},
		{loadpoint.PollAlways, api.StatusB, tRefresh, true},
		{loadpoint.PollAlways, api.StatusC, -1, true},
		{loadpoint.PollAlways, api.StatusC, 0, true},
		{loadpoint.PollAlways, api.StatusC, tNoRefresh, true}, // cached by vehicle
		{loadpoint.PollAlways, api.StatusC, tRefresh, true},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		if tc.dt < 0 {
			lp.socUpdated = time.Time{}
		} else {
			clock.Add(tc.dt)
		}

		lp.Soc.Poll.Mode = tc.mode
		lp.status = tc.status

		res := lp.vehicleSocPollAllowed()
		if res {
			// mimic update outside of socPollAllowed
			lp.socUpdated = clock.Now()
		}

		assert.Equal(t, tc.res, res)
	}
}

// test PV hysteresis after phase switch down, depending on remaining energy
func TestPVHysteresisAfterPhaseSwitch(t *testing.T) {
	const dt = time.Minute
	type se struct {
		site    float64
		delay   time.Duration // test case delay since start
		current float64
	}
	tc := []struct {
		series []se
	}{
		// immediately disable when threshold met after phase switch
		{[]se{
			{1200, 0, minA},
			{1200, 1, minA},
			{1200, dt - 1, minA},
			{1200, dt + 1, 0},
		}},
		// stay enabled when threshold not met after phase switch
		{[]se{
			{500, 0, minA},
			{500, 1, minA},
			{500, dt - 1, minA},
			{500, dt + 1, minA},
		}},
	}

	for _, tc := range tc {
		t.Log(tc)

		clock := clock.NewMock()
		ctrl := gomock.NewController(t)

		switcher := api.NewMockPhaseSwitcher(ctrl)
		switcher.EXPECT().Phases1p3p(1)

		charger := struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{
			api.NewMockCharger(ctrl), switcher,
		}

		Voltage = 100
		lp := &Loadpoint{
			log:         util.NewLogger("foo"),
			wakeUpTimer: NewTimer(),
			clock:       clock,
			charger:     charger,
			minCurrent:  minA,
			maxCurrent:  maxA,
			solarShare:  1,
			Disable: loadpoint.ThresholdConfig{
				Delay: dt,
			},
			status:      api.StatusC,
			enabled:     true,
			chargePower: 3 * Voltage * minA, // charging 3p at min current
		}

		start := clock.Now()

		for step, se := range tc.series {
			clock.Set(start.Add(se.delay))
			assert.Equal(t, se.current, lp.pvMaxCurrent(se.site, 0, false, false), step)
		}

		ctrl.Finish()
	}
}

func TestConnectionDurationDropDetection(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	ch := api.NewMockCharger(ctrl)
	ct := api.NewMockConnectionTimer(ctrl)

	charger := struct {
		api.Charger
		api.ConnectionTimer
	}{
		ch, ct,
	}

	ch.EXPECT().Status().Return(api.StatusB, nil)
	ch.EXPECT().Enabled().AnyTimes().Return(true, nil)
	ch.EXPECT().MaxCurrent(int64(minA)).Return(nil)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		charger:     charger,
		minCurrent:  minA,
		maxCurrent:  maxA,
		chargeMeter: &Null{},    // silence nil panics
		chargeRater: &Null{},    // silence nil panics
		chargeTimer: &Null{},    // silence nil panics
		wakeUpTimer: NewTimer(), // silence nil panics
	}

	attachListeners(t, lp)

	connectedTime := clock.Now().Add(-10 * time.Minute)

	lp.enabled = true
	lp.status = api.StatusC
	lp.connectedDuration = 10 * time.Minute
	lp.connectedTime = connectedTime

	ct.EXPECT().ConnectionDuration().Return(0*time.Second, nil)
	lp.Update(500, 0, nil, nil, false, false, 0, nil, nil, nil)
	ctrl.Finish()

	assert.NotEqual(t, connectedTime, lp.connectedTime)
}

func TestWelcomeChargeAppliedOnlyOnce(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	ch := api.NewMockCharger(ctrl)
	fd := api.NewMockFeatureDescriber(ctrl)

	charger := struct {
		api.Charger
		api.FeatureDescriber
	}{
		ch, fd,
	}

	ch.EXPECT().Enabled().AnyTimes().Return(true, nil)
	ch.EXPECT().MaxCurrent(int64(minA)).Return(nil)
	fd.EXPECT().Features().AnyTimes().Return([]api.Feature{
		api.WelcomeCharge,
	})

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		charger:     charger,
		minCurrent:  minA,
		maxCurrent:  maxA,
		chargeMeter: &Null{},    // silence nil panics
		chargeRater: &Null{},    // silence nil panics
		chargeTimer: &Null{},    // silence nil panics
		wakeUpTimer: NewTimer(), // silence nil panics
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.status = api.StatusA

	// No welcome charge when not connected
	ch.EXPECT().Status().Return(api.StatusA, nil)
	welcomeCharge, _ := lp.updateChargerStatus()
	assert.False(t, welcomeCharge)

	// Welcome charge when connected
	ch.EXPECT().Status().Return(api.StatusC, nil)
	welcomeCharge, _ = lp.updateChargerStatus()
	assert.True(t, welcomeCharge)

	// No welcome charge when still connected
	ch.EXPECT().Status().Return(api.StatusB, nil)
	welcomeCharge, _ = lp.updateChargerStatus()
	assert.False(t, welcomeCharge)
}

// TestBatteryBoostHold verifies that in the hold state (soc limit reached) battery
// boost no longer draws power from the battery, while still counting as active so
// the site keeps prioritising the vehicle over recharging the battery (#30558).
func TestBatteryBoostHold(t *testing.T) {
	lp := &Loadpoint{log: util.NewLogger("foo")}

	// disabled draws nothing
	lp.batteryBoost = boostDisabled
	assert.Equal(t, 0.0, lp.boostPower(2000), "disabled")

	// hold draws nothing (stops draining the battery)...
	lp.batteryBoost = boostHold
	assert.Equal(t, 0.0, lp.boostPower(2000), "hold")

	// ...but is still an active boost state (GetBatteryBoost != boostDisabled),
	// which is what keeps the sitePower priority adjustment applied to the loadpoint
	assert.NotEqual(t, boostDisabled, lp.GetBatteryBoost(), "hold is active")
}

// TestPVSolarShare verifies the pv enable/disable points derived from solarShare
// and that manually configured thresholds take precedence over the solar share.
func TestPVSolarShare(t *testing.T) {
	Voltage = 100
	ctrl := gomock.NewController(t)

	newLp := func(share float64, enabled bool, enableT, disableT float64) *Loadpoint {
		lp := &Loadpoint{
			log:            util.NewLogger("foo"),
			clock:          clock.NewMock(),
			charger:        api.NewMockCharger(ctrl),
			minCurrent:     minA,
			maxCurrent:     maxA,
			phases:         3,
			measuredPhases: 3,
			Enable:         loadpoint.ThresholdConfig{Threshold: enableT},
			Disable:        loadpoint.ThresholdConfig{Threshold: disableT},
			solarShare:     share,
		}
		lp.status = api.StatusC
		lp.enabled = enabled
		return lp
	}

	minPower := currentToPower(minA, 3)

	// enable: share 1.0 requires the full min power as surplus
	assert.Equal(t, minA, newLp(1, false, 0, 0).pvMaxCurrent(-minPower, 0, false, false),
		"should enable at full surplus")
	assert.Equal(t, 0.0, newLp(1, false, 0, 0).pvMaxCurrent(-minPower+100, 0, false, false),
		"must not enable below full surplus")

	// enable: share 0.5 accepts half the min power from grid
	assert.Equal(t, minA, newLp(0.5, false, 0, 0).pvMaxCurrent(-minPower/2, 0, false, false),
		"should enable at half surplus")

	// enable: share 0 starts as soon as there is no grid import
	assert.Equal(t, minA, newLp(0, false, 0, 0).pvMaxCurrent(0, 0, false, false),
		"should enable without surplus")

	// disable: share 1.0 derives threshold 0, so feed-in keeps charging
	assert.Equal(t, minA, newLp(1, true, 0, 0).pvMaxCurrent(-500, 0, false, false),
		"must not disable while feeding in")
	assert.Equal(t, 0.0, newLp(1, true, 0, 0).pvMaxCurrent(100, 0, false, false),
		"should disable on grid draw")

	// disable: share 0.5 tolerates grid import up to half the min power
	assert.Equal(t, minA, newLp(0.5, true, 0, 0).pvMaxCurrent(minPower/2-100, 0, false, false),
		"must not disable within allowed grid import")
	assert.Equal(t, 0.0, newLp(0.5, true, 0, 0).pvMaxCurrent(minPower/2+100, 0, false, false),
		"should disable above allowed grid import")

	// configured thresholds take precedence over solar share
	assert.Equal(t, minA, newLp(1, false, 5000, 0).pvMaxCurrent(4000, 0, false, false),
		"enable threshold should apply despite solar share")
	assert.Equal(t, minA, newLp(1, true, 0, 5000).pvMaxCurrent(100, 0, false, false),
		"disable threshold should apply despite solar share")
}

// TestPVSolarSharePhases verifies that the derived switch points scale with the
// phases charging actually runs on, not with the theoretical 1p minimum.
func TestPVSolarSharePhases(t *testing.T) {
	Voltage = 100
	ctrl := gomock.NewController(t)

	// 1p3p charger pinned to 3p cannot scale down, so the enable point must
	// require the full 3p min power - not the 1p min power of EffectiveMinPower
	charger := struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{
		api.NewMockCharger(ctrl), api.NewMockPhaseSwitcher(ctrl),
	}

	newLp := func(enabled bool) *Loadpoint {
		lp := &Loadpoint{
			log:              util.NewLogger("foo"),
			clock:            clock.NewMock(),
			charger:          charger,
			minCurrent:       minA,
			maxCurrent:       maxA,
			phases:           3,
			phasesConfigured: 3,
			solarShare:       1,
		}
		lp.status = api.StatusC
		lp.enabled = enabled
		return lp
	}

	assert.Equal(t, 0.0, newLp(false).pvMaxCurrent(-currentToPower(minA, 1), 0, false, false),
		"must not enable at 1p surplus while pinned to 3p")
	assert.Equal(t, minA, newLp(false).pvMaxCurrent(-currentToPower(minA, 3), 0, false, false),
		"should enable at 3p surplus")
}

// default vehicle referencing a disabled vehicle must not fail loadpoint creation
func TestNewLoadpointFromConfigDisabledVehicle(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)

	// disabled vehicle: device registered without instance
	var vehicle api.Vehicle
	require.NoError(t, config.Vehicles().Add(config.NewStaticDevice(config.Named{Name: "vehicle"}, vehicle)))
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{Name: "charger"}, api.Charger(api.NewMockCharger(ctrl)))))

	lp, err := NewLoadpointFromConfig(util.NewLogger("foo"), nil, nil, map[string]any{
		"charger": "charger",
		"vehicle": "vehicle",
	})
	require.NoError(t, err)
	require.Nil(t, lp.defaultVehicle)

	// disabled vehicle is filtered from instances
	require.Empty(t, config.Instances(config.Vehicles().Devices()))
}

// TestPVDisableContinuousDeviceShortfall is a regression test for #32282: a continuous
// device consuming less than its min power demand keeps the remainder out of site
// power, so the disable gate never trips on the missing surplus. The shortfall
// towards min power is projected into the gate regardless of charge status, since
// some chargers (sgready) report StatusC while the device is idle.
func TestPVDisableContinuousDeviceShortfall(t *testing.T) {
	const dt = time.Minute

	tc := []struct {
		name        string
		status      api.ChargeStatus
		chargePower float64
		site        float64
		current     float64
	}{
		// idle device, export below its 600W demand: disable
		{"idle, insufficient surplus", api.StatusB, 0, -400, 0},
		// idle device, export covers its demand: keep enabled
		{"idle, sufficient surplus", api.StatusB, 0, -700, 7},
		// idle device reporting StatusC (sgready-style): still disable
		{"idle, StatusC, insufficient surplus", api.StatusC, 0, -400, 0},
		// starting device: own draw plus surplus covers min power, keep enabled
		{"starting, demand covered", api.StatusB, 300, -400, 7},
		// running below min power without surplus for the remaining demand: disable
		{"running below min, insufficient surplus", api.StatusC, 300, -200, 0},
		// consuming min power, importing: unchanged disable behavior
		{"consuming, importing", api.StatusC, 600, 100, 0},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			clock := clock.NewMock()

			Voltage = 100
			lp := &Loadpoint{
				log:              util.NewLogger("foo"),
				clock:            clock,
				charger:          &continuousCharger{},
				minCurrent:       minA,
				maxCurrent:       maxA,
				phases:           1,
				phasesConfigured: 1,
				measuredPhases:   1,
				status:           tc.status,
				enabled:          true,
				chargePower:      tc.chargePower,
				solarShare:       1,
				Disable:          loadpoint.ThresholdConfig{Delay: dt},
			}

			start := clock.Now()
			for _, delay := range []time.Duration{0, dt + 1} {
				clock.Set(start.Add(delay))
				current := lp.pvMaxCurrent(tc.site, 0, false, false)

				// before the disable delay elapses the device keeps running
				if delay == 0 {
					assert.Equal(t, max(tc.current, minA), current, "before disable delay")
					continue
				}

				assert.Equal(t, tc.current, current, "after disable delay")
			}
		})
	}
}
