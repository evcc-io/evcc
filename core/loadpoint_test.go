package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
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

func createChannels(t *testing.T) (chan util.Param, chan push.Event, chan *Loadpoint) {
	t.Helper()

	uiChan := make(chan util.Param)
	pushChan := make(chan push.Event)
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

func attachChannels(lp *Loadpoint, uiChan chan util.Param, pushChan chan push.Event, lpChan chan *Loadpoint) {
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
	lp.Prepare(uiChan, pushChan, lpChan)
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
		status api.ChargeStatus
		mode   api.ChargeMode
		expect func(h *api.MockCharger)
	}{
		{api.StatusA, api.ModeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusA, api.ModeNow, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusA, api.ModeMinPV, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusA, api.ModePV, func(h *api.MockCharger) {
			h.EXPECT().Enable(false) // zero since update called with 0
		}},

		{api.StatusB, api.ModeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusB, api.ModeNow, func(h *api.MockCharger) {
			h.EXPECT().MaxCurrent(int64(maxA)) // true
		}},
		{api.StatusB, api.ModeMinPV, func(h *api.MockCharger) {
			// MaxCurrent omitted since identical value
		}},
		{api.StatusB, api.ModePV, func(h *api.MockCharger) {
			// zero since update called with 0
			// force = false due to pv mode climater check
			h.EXPECT().Enable(false)
		}},

		{api.StatusC, api.ModeOff, func(h *api.MockCharger) {
			h.EXPECT().Enable(false)
		}},
		{api.StatusC, api.ModeNow, func(h *api.MockCharger) {
			h.EXPECT().MaxCurrent(int64(maxA)) // true
		}},
		{api.StatusC, api.ModeMinPV, func(h *api.MockCharger) {
			// MaxCurrent omitted since identical value
		}},
		{api.StatusC, api.ModePV, func(h *api.MockCharger) {
			// omitted since PV balanced
		}},
	}

	for _, tc := range tc {
		t.Log(tc)

		clck := clock.NewMock()
		ctrl := gomock.NewController(t)
		charger := api.NewMockCharger(ctrl)

		lp := &Loadpoint{
			log:           util.NewLogger("foo"),
			bus:           evbus.New(),
			clock:         clck,
			charger:       charger,
			chargeMeter:   &Null{}, // silence nil panics
			chargeRater:   &Null{}, // silence nil panics
			chargeTimer:   &Null{}, // silence nil panics
			wakeUpTimer:   NewTimer(),
			sessionEnergy: NewEnergyMetrics(),
			minCurrent:    minA,
			maxCurrent:    maxA,
			phases:        1,
			status:        tc.status, // no status change
		}

		attachListeners(t, lp)

		// initial status
		charger.EXPECT().Status().Return(tc.status, nil)
		charger.EXPECT().Enabled().Return(true, nil)

		if tc.expect != nil {
			tc.expect(charger)
		}

		lp.mode = tc.mode
		lp.Update(0, false, false, false, 0, nil, nil) // false,sitePower false,0

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

			clck := clock.NewMock()
			ctrl := gomock.NewController(t)
			charger := api.NewMockCharger(ctrl)

			Voltage = 100
			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				clock:          clck,
				charger:        charger,
				minCurrent:     minA,
				maxCurrent:     maxA,
				phases:         phases,
				measuredPhases: phases,
				Enable: ThresholdConfig{
					Threshold: tc.enable,
					Delay:     dt,
				},
				Disable: ThresholdConfig{
					Threshold: tc.disable,
					Delay:     dt,
				},
			}

			// charging, otherwise PV mode logic is short-circuited
			lp.status = status

			start := clck.Now()

			for step, se := range tc.series {
				clck.Set(start.Add(se.delay))

				// maxCurrent will read actual current and enabled state in PV mode
				// charger.EXPECT().Enabled().Return(tc.enabled, nil)

				lp.enabled = tc.enabled
				current := lp.pvMaxCurrent(api.ModePV, se.site, false, false)

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

	clck := clock.NewMock()
	ctrl := gomock.NewController(t)

	Voltage = 100
	lp := &Loadpoint{
		log:            util.NewLogger("foo"),
		clock:          clck,
		minCurrent:     minA,
		maxCurrent:     maxA,
		phases:         phases,
		measuredPhases: phases,
	}

	// not connected, test PV mode logic  short-circuited
	lp.status = api.StatusA

	// maxCurrent will read enabled state in PV mode
	sitePower := -float64(phases)*minA*Voltage + 1 // 1W below min power
	current := lp.pvMaxCurrent(api.ModePV, sitePower, false, false)

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

	socEstimator := soc.NewEstimator(util.NewLogger("foo"), charger, vehicle, false)

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
		minCurrent:    minA,
		maxCurrent:    maxA,
		vehicle:       vehicle,      // needed for targetSoc check
		socEstimator:  socEstimator, // instead of vehicle: vehicle,
		mode:          api.ModeNow,
		sessionEnergy: NewEnergyMetrics(),
		limitSoc:      90, // session limit
		Soc: SocConfig{
			Poll: PollConfig{
				Mode:     pollConnected, // allow polling when connected
				Interval: pollInterval,
			},
		},
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.chargeCurrent = minA
	lp.status = api.StatusC

	t.Log("charging below soc target")
	vehicle.EXPECT().Soc().Return(85.0, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)
	lp.Update(500, false, false, false, 0, nil, nil)
	ctrl.Finish()

	t.Log("charging above target - soc deactivates charger")
	clock.Add(5 * time.Minute)
	vehicle.EXPECT().Soc().Return(90.0, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Enable(false).Return(nil)
	lp.Update(500, false, false, false, 0, nil, nil)
	ctrl.Finish()

	t.Log("deactivated charger changes status to B")
	clock.Add(5 * time.Minute)
	vehicle.EXPECT().Soc().Return(95.0, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	lp.Update(-5000, false, false, false, 0, nil, nil)
	ctrl.Finish()

	t.Log("soc has fallen below target - soc update prevented by timer")
	clock.Add(5 * time.Minute)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	lp.Update(-5000, false, false, false, 0, nil, nil)
	ctrl.Finish()

	t.Log("soc has fallen below target - soc update timer expired")
	clock.Add(pollInterval)
	vehicle.EXPECT().Soc().Return(85.0, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Enable(true).Return(nil)
	lp.Update(-5000, false, false, false, 0, nil, nil)
	ctrl.Finish()
}

func TestSetModeAndSocAtDisconnect(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:           util.NewLogger("foo"),
		bus:           evbus.New(),
		clock:         clock,
		charger:       charger,
		chargeMeter:   &Null{}, // silence nil panics
		chargeRater:   &Null{}, // silence nil panics
		chargeTimer:   &Null{}, // silence nil panics
		wakeUpTimer:   NewTimer(),
		sessionEnergy: NewEnergyMetrics(),
		minCurrent:    minA,
		maxCurrent:    maxA,
		status:        api.StatusC,
		Mode_:         api.ModeOff, // default mode
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.chargeCurrent = minA
	lp.mode = api.ModeNow

	t.Log("charging at min")
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	charger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)
	lp.Update(500, false, false, false, 0, nil, nil)

	t.Log("switch off when disconnected")
	clock.Add(5 * time.Minute)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusA, nil)
	charger.EXPECT().Enable(false).Return(nil)
	lp.Update(-3000, false, false, false, 0, nil, nil)

	if mode := lp.GetMode(); mode != api.ModeOff {
		t.Error("unexpected mode", mode)
	}

	ctrl.Finish()
}

// cacheExpecter can be used to verify asynchronously written values from cache
func cacheExpecter(t *testing.T, lp *Loadpoint) (*util.Cache, func(key string, val interface{})) {
	t.Helper()

	// attach cache for verifying values
	paramC := make(chan util.Param)
	lp.uiChan = paramC

	cache := util.NewCache()
	go cache.Run(paramC)

	expect := func(key string, val interface{}) {
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
		log:           util.NewLogger("foo"),
		bus:           evbus.New(),
		clock:         clock,
		charger:       charger,
		chargeMeter:   &Null{}, // silence nil panics
		chargeRater:   rater,
		chargeTimer:   &Null{}, // silence nil panics
		wakeUpTimer:   NewTimer(),
		sessionEnergy: NewEnergyMetrics(),
		minCurrent:    minA,
		maxCurrent:    maxA,
		status:        api.StatusC,
	}

	attachListeners(t, lp)

	lp.enabled = true
	lp.chargeCurrent = maxA
	lp.mode = api.ModeNow

	// attach cache for verifying values
	_, expectCache := cacheExpecter(t, lp)

	t.Log("start charging at 0 kWh")
	rater.EXPECT().ChargedEnergy().Return(0.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, false, false, false, 0, nil, nil)

	t.Log("at 1:00h charging at 5 kWh")
	clock.Add(time.Hour)
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, false, false, false, 0, nil, nil)
	expectCache("chargedEnergy", 5000.0)

	t.Log("at 1:00h stop charging at 5 kWh")
	clock.Add(time.Second)
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	lp.Update(-1, false, false, false, 0, nil, nil)
	expectCache("chargedEnergy", 5000.0)

	t.Log("at 1:00h restart charging at 5 kWh")
	clock.Add(time.Second)
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, false, false, false, 0, nil, nil)
	expectCache("chargedEnergy", 5000.0)

	t.Log("at 1:30h continue charging at 7.5 kWh")
	clock.Add(30 * time.Minute)
	rater.EXPECT().ChargedEnergy().Return(7.5, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.Update(-1, false, false, false, 0, nil, nil)
	expectCache("chargedEnergy", 7500.0)

	t.Log("at 2:00h stop charging at 10 kWh")
	clock.Add(30 * time.Minute)
	rater.EXPECT().ChargedEnergy().Return(10.0, nil)
	charger.EXPECT().Enabled().Return(lp.enabled, nil)
	charger.EXPECT().Status().Return(api.StatusB, nil)
	lp.Update(-1, false, false, false, 0, nil, nil)
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
		Soc: SocConfig{
			Poll: PollConfig{
				Interval: time.Hour,
			},
		},
	}

	tc := []struct {
		mode   string
		status api.ChargeStatus
		dt     time.Duration
		res    bool
	}{
		// pollCharging
		{pollCharging, api.StatusA, -1, false},
		{pollCharging, api.StatusA, 0, false},
		{pollCharging, api.StatusA, tRefresh, false},
		{pollCharging, api.StatusB, -1, true}, // poll once when car connected
		{pollCharging, api.StatusB, 0, false},
		{pollCharging, api.StatusB, tRefresh, false},
		{pollCharging, api.StatusC, -1, true},
		{pollCharging, api.StatusC, 0, true},
		{pollCharging, api.StatusC, tNoRefresh, true}, // cached by vehicle
		{pollCharging, api.StatusB, -1, true},         // fetch if car stopped charging
		{pollCharging, api.StatusB, 0, false},         // no more polling
		{pollCharging, api.StatusB, tRefresh, false},  // no more polling

		// pollConnected
		{pollConnected, api.StatusA, -1, false},
		{pollConnected, api.StatusA, 0, false},
		{pollConnected, api.StatusA, tRefresh, false},
		{pollConnected, api.StatusB, -1, true},
		{pollConnected, api.StatusB, 0, false},
		{pollConnected, api.StatusB, tNoRefresh, false},
		{pollConnected, api.StatusB, tRefresh, true},
		{pollConnected, api.StatusC, -1, true},
		{pollConnected, api.StatusC, 0, true},
		{pollConnected, api.StatusC, tNoRefresh, true}, // cached by vehicle
		{pollConnected, api.StatusC, tRefresh, true},

		// pollAlways
		{pollAlways, api.StatusA, -1, true},
		{pollAlways, api.StatusA, 0, false},
		{pollAlways, api.StatusA, tNoRefresh, false},
		{pollAlways, api.StatusA, tRefresh, true},
		{pollAlways, api.StatusB, -1, true},
		{pollAlways, api.StatusB, 0, false},
		{pollAlways, api.StatusB, tNoRefresh, false},
		{pollAlways, api.StatusB, tRefresh, true},
		{pollAlways, api.StatusC, -1, true},
		{pollAlways, api.StatusC, 0, true},
		{pollAlways, api.StatusC, tNoRefresh, true}, // cached by vehicle
		{pollAlways, api.StatusC, tRefresh, true},
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

		clck := clock.NewMock()
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
			log:        util.NewLogger("foo"),
			clock:      clck,
			charger:    charger,
			minCurrent: minA,
			maxCurrent: maxA,
			Disable: ThresholdConfig{
				Delay: dt,
			},
			status:  api.StatusC,
			enabled: true,
		}

		start := clck.Now()

		for step, se := range tc.series {
			clck.Set(start.Add(se.delay))
			assert.Equal(t, se.current, lp.pvMaxCurrent(api.ModePV, se.site, false, false), step)
		}

		ctrl.Finish()
	}
}
