package core

import (
	"testing"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"
	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
)

const (
	lpMinCurrent int64 = 6
	lpMaxCurrent int64 = 16
)

type Null struct{}

func (n *Null) CurrentPower() (float64, error) {
	return 0, nil
}

func (n *Null) ChargedEnergy() (float64, error) {
	return 0, nil
}

func (n *Null) ChargingTime() (time.Duration, error) {
	return 0, nil
}

func attachListeners(t *testing.T, lp *LoadPoint) {
	uiChan := make(chan util.Param)
	pushChan := make(chan push.Event)
	lpChan := make(chan *LoadPoint)

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

	lp.Prepare(uiChan, pushChan, lpChan)
}

func TestNew(t *testing.T) {
	lp := NewLoadPoint(util.NewLogger("foo"))

	if lp.Phases != 1 {
		t.Errorf("Phases %v", lp.Phases)
	}
	if lp.MinCurrent != lpMinCurrent {
		t.Errorf("MinCurrent %v", lp.MinCurrent)
	}
	if lp.MaxCurrent != lpMaxCurrent {
		t.Errorf("MaxCurrent %v", lp.MaxCurrent)
	}
	if lp.Sensitivity != 10 {
		t.Errorf("Sensitivity %v", lp.Sensitivity)
	}
	if lp.status != api.StatusNone {
		t.Errorf("status %v", lp.status)
	}
	if lp.charging {
		t.Errorf("charging %v", lp.charging)
	}
}

func TestUpdate(t *testing.T) {
	tc := []struct {
		status api.ChargeStatus
		mode   api.ChargeMode
		expect func(h *mock.MockHandler)
	}{
		{api.StatusA, api.ModeOff, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0))
		}},
		{api.StatusA, api.ModeNow, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0))
		}},
		{api.StatusA, api.ModeMinPV, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0))
		}},
		{api.StatusA, api.ModePV, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0)) // zero since update called with 0
			// h.EXPECT().Enabled().Return(false) // short-circuited due to status != C
		}},

		{api.StatusB, api.ModeOff, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0), true)
		}},
		{api.StatusB, api.ModeNow, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(lpMaxCurrent, true)
		}},
		{api.StatusB, api.ModeMinPV, func(h *mock.MockHandler) {
			// min since update called with 0
			// force = false due to pv mode climater check
			h.EXPECT().Ramp(lpMinCurrent, false)
		}},
		{api.StatusB, api.ModePV, func(h *mock.MockHandler) {
			// zero since update called with 0
			// force = false due to pv mode climater check
			h.EXPECT().Ramp(int64(0), false)
			// h.EXPECT().Enabled().Return(false) // short-circuited due to status != C
		}},

		{api.StatusC, api.ModeOff, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0), true)
		}},
		{api.StatusC, api.ModeNow, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(lpMaxCurrent, true)
		}},
		{api.StatusC, api.ModeMinPV, func(h *mock.MockHandler) {
			// min since update called with 0
			// force = false due to pv mode climater check
			h.EXPECT().Ramp(lpMinCurrent, false)
		}},
		{api.StatusC, api.ModePV, func(h *mock.MockHandler) {
			// zero since update called with 0
			// force = false due to pv mode climater check
			h.EXPECT().Ramp(int64(0), false)
			h.EXPECT().Enabled().Return(false)
		}},
	}

	for _, tc := range tc {
		t.Log(tc)

		clck := clock.NewMock()
		ctrl := gomock.NewController(t)
		handler := mock.NewMockHandler(ctrl)

		lp := &LoadPoint{
			log:         util.NewLogger("foo"),
			bus:         evbus.New(),
			clock:       clck,
			chargeMeter: &Null{}, // silence nil panics
			chargeRater: &Null{}, // silence nil panics
			chargeTimer: &Null{}, // silence nil panics
			HandlerConfig: HandlerConfig{
				MinCurrent: lpMinCurrent,
				MaxCurrent: lpMaxCurrent,
			},
			handler: handler,
			status:  tc.status, // no status change
		}

		handler.EXPECT().Prepare().Return()
		attachListeners(t, lp)

		handler.EXPECT().Status().Return(tc.status, nil)
		handler.EXPECT().TargetCurrent().Return(int64(0))

		if tc.status != api.StatusA {
			handler.EXPECT().SyncEnabled()

			if tc.mode == api.ModeMinPV || tc.mode == api.ModePV {
				handler.EXPECT().TargetCurrent().Return(int64(0))
			}
		}

		if tc.expect != nil {
			tc.expect(handler)
		}

		lp.Mode = tc.mode
		lp.Update(0)

		ctrl.Finish()
	}
}

func TestPVHysteresisForStatusC(t *testing.T) {
	dt := time.Minute
	type se struct {
		site    float64
		delay   time.Duration // test case delay since start
		current int64
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
			{-6 * 100 * 10, 0, 0},
			{-6 * 100 * 10, 1, 0},
			{-6 * 100 * 10, dt - 1, 0},
			{-6 * 100 * 10, dt + 1, lpMinCurrent},
		}},
		// keep disabled when threshold not configured
		{false, 0, 0, []se{
			{-400, 0, 0},
			{-400, 1, 0},
			{-400, dt - 1, 0},
			{-400, dt + 1, 0},
		}},
		// keep disabled when threshold not met
		{false, -500, 0, []se{
			{-400, 0, 0},
			{-400, 1, 0},
			{-400, dt - 1, 0},
			{-400, dt + 1, 0},
		}},
		// enable when threshold met
		{false, -500, 0, []se{
			{-500, 0, 0},
			{-500, 1, 0},
			{-500, dt - 1, 0},
			{-500, dt + 1, lpMinCurrent},
		}},
		// keep enabled at max
		{true, 500, 0, []se{
			{-16 * 100 * 10, 0, lpMaxCurrent},
			{-16 * 100 * 10, 1, lpMaxCurrent},
			{-16 * 100 * 10, dt - 1, lpMaxCurrent},
			{-16 * 100 * 10, dt + 1, lpMaxCurrent},
		}},
		// keep enabled at min
		{true, 500, 0, []se{
			{-6 * 100 * 10, 0, lpMinCurrent},
			{-6 * 100 * 10, 1, lpMinCurrent},
			{-6 * 100 * 10, dt - 1, lpMinCurrent},
			{-6 * 100 * 10, dt + 1, lpMinCurrent},
		}},
		// keep enabled at min (negative threshold)
		{true, 0, 500, []se{
			{-500, 0, lpMinCurrent},
			{-500, 1, lpMinCurrent},
			{-500, dt - 1, lpMinCurrent},
			{-500, dt + 1, lpMinCurrent},
		}},
		// disable when threshold met
		{true, 0, 500, []se{
			{500, 0, lpMinCurrent},
			{500, 1, lpMinCurrent},
			{500, dt - 1, lpMinCurrent},
			{500, dt + 1, 0},
		}},
		// reset enable timer when threshold not met while timer active
		{false, -500, 0, []se{
			{-500, 0, 0},
			{-500, 1, 0},
			{-499, dt - 1, 0}, // should reset timer
			{-500, dt + 1, 0}, // new begin of timer
			{-500, 2*dt - 2, 0},
			{-500, 2*dt - 1, lpMinCurrent},
		}},
		// reset enable timer when threshold not met while timer active and threshold not configured
		{false, 0, 0, []se{
			{-6*100*10 - 1, dt + 1, 0},
			{-6 * 100 * 10, dt + 1, 0},
			{-6 * 100 * 10, dt + 2, 0},
			{-6 * 100 * 10, 2 * dt, 0},
			{-6 * 100 * 10, 2*dt + 2, lpMinCurrent},
		}},
		// reset disable timer when threshold not met while timer active
		{true, 0, 500, []se{
			{500, 0, lpMinCurrent},
			{500, 1, lpMinCurrent},
			{499, dt - 1, lpMinCurrent},   // reset timer
			{500, dt + 1, lpMinCurrent},   // within reset timer duration
			{500, 2*dt - 2, lpMinCurrent}, // still within reset timer duration
			{500, 2*dt - 1, 0},            // reset timer elapsed
		}},
	}

	for _, tc := range tc {
		t.Log(tc)

		clck := clock.NewMock()
		ctrl := gomock.NewController(t)
		handler := mock.NewMockHandler(ctrl)

		Voltage = 100
		lp := &LoadPoint{
			log:   util.NewLogger("foo"),
			clock: clck,
			HandlerConfig: HandlerConfig{
				MinCurrent: lpMinCurrent,
				MaxCurrent: lpMaxCurrent,
			},
			handler: handler,
			Phases:  10,
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
		lp.status = api.StatusC

		start := clck.Now()

		for step, se := range tc.series {
			clck.Set(start.Add(se.delay))

			// maxCurrent will read actual current and enabled state in PV mode
			handler.EXPECT().TargetCurrent().Return(int64(0))
			handler.EXPECT().Enabled().Return(tc.enabled)

			current := lp.pvMaxCurrent(api.ModePV, se.site)

			if current != se.current {
				t.Errorf("step %d: wanted %d, got %d", step, se.current, current)
			}
		}

		ctrl.Finish()
	}
}

func TestPVHysteresisForStatusOtherThanC(t *testing.T) {
	clck := clock.NewMock()
	ctrl := gomock.NewController(t)
	handler := mock.NewMockHandler(ctrl)

	Voltage = 100
	lp := &LoadPoint{
		log:   util.NewLogger("foo"),
		clock: clck,
		HandlerConfig: HandlerConfig{
			MinCurrent: lpMinCurrent,
			MaxCurrent: lpMaxCurrent,
		},
		handler: handler,
		Phases:  10,
	}

	// not connected, test PV mode logic  short-circuited
	lp.status = api.StatusA

	// maxCurrent will read actual current in PV mode
	handler.EXPECT().TargetCurrent().Return(int64(0))

	// maxCurrent will read enabled state in PV mode
	sitePower := -float64(minA*lp.Phases)*Voltage + 1 // 1W below min power
	current := lp.pvMaxCurrent(api.ModePV, sitePower)

	if current != 0 {
		t.Errorf("PV mode could not disable charger as expected. Expected 0, got %d", current)
	}

	ctrl.Finish()
}

func TestDisableAndEnableAtTargetSoC(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	handler := mock.NewMockHandler(ctrl)
	vehicle := mock.NewMockVehicle(ctrl)

	// wrap vehicle with estimator
	vehicle.EXPECT().Capacity().Return(int64(10))
	socEstimator := wrapper.NewSocEstimator(util.NewLogger("foo"), vehicle, false)

	lp := &LoadPoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		chargeMeter: &Null{}, // silence nil panics
		chargeRater: &Null{}, // silence nil panics
		chargeTimer: &Null{}, // silence nil panics
		HandlerConfig: HandlerConfig{
			MinCurrent: lpMinCurrent,
			MaxCurrent: lpMaxCurrent,
		},
		handler:      handler,
		vehicle:      vehicle,      // needed for targetSoC check
		socEstimator: socEstimator, // instead of vehicle: vehicle,
		status:       api.StatusC,
		Mode:         api.ModeNow,
		SoC: SoCConfig{
			Target: 90,
		},
	}

	handler.EXPECT().Prepare().Return()
	attachListeners(t, lp)

	// charging below target
	handler.EXPECT().TargetCurrent().Return(int64(6))
	handler.EXPECT().Status().Return(api.StatusC, nil)
	vehicle.EXPECT().ChargeState().Return(85.0, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(500)

	// charging above target deactivates charger
	clock.Add(5 * time.Minute)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	handler.EXPECT().Status().Return(api.StatusC, nil)
	vehicle.EXPECT().ChargeState().Return(90.0, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().Ramp(int64(0), true).Return(nil) // true due to immediately handling climate requests
	lp.Update(500)

	// deactivated charger changes status to B
	clock.Add(5 * time.Minute)
	handler.EXPECT().TargetCurrent().Return(int64(0))
	handler.EXPECT().Status().Return(api.StatusB, nil)
	handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	vehicle.EXPECT().ChargeState().Return(95.0, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().Ramp(int64(0), true).Return(nil) // true due to immediately handling climate requests
	lp.Update(-5000)

	// soc has fallen below target
	clock.Add(5 * time.Minute)
	handler.EXPECT().TargetCurrent().Return(int64(0))
	handler.EXPECT().Status().Return(api.StatusB, nil)
	vehicle.EXPECT().ChargeState().Return(85.0, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().Ramp(int64(16), true).Return(nil) // TODO don't treat this as forced change
	lp.Update(-5000)

	ctrl.Finish()
}

func TestSetModeAndSocAtDisconnect(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	handler := mock.NewMockHandler(ctrl)

	lp := &LoadPoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		chargeMeter: &Null{}, // silence nil panics
		chargeRater: &Null{}, // silence nil panics
		chargeTimer: &Null{}, // silence nil panics
		HandlerConfig: HandlerConfig{
			MinCurrent: lpMinCurrent,
			MaxCurrent: lpMaxCurrent,
		},
		handler: handler,
		status:  api.StatusC,
		OnDisconnect: struct {
			Mode      api.ChargeMode `mapstructure:"mode"`      // Charge mode to apply when car disconnected
			TargetSoC int            `mapstructure:"targetSoC"` // Target SoC to apply when car disconnected
		}{
			Mode:      api.ModeOff,
			TargetSoC: 70,
		},
	}

	handler.EXPECT().Prepare().Return()
	attachListeners(t, lp)

	lp.Mode = api.ModeNow
	handler.EXPECT().TargetCurrent().Return(int64(6))
	handler.EXPECT().Status().Return(api.StatusC, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(500)

	clock.Add(5 * time.Minute)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	handler.EXPECT().Status().Return(api.StatusA, nil)
	handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	handler.EXPECT().Ramp(int64(0)).Return(nil)
	lp.Update(-3000)

	if lp.Mode != api.ModeOff {
		t.Error("unexpected mode", lp.Mode)
	}

	ctrl.Finish()
}

// cacheExpecter can be used to verify asynchronously written values from cache
func cacheExpecter(t *testing.T, lp *LoadPoint) (*util.Cache, func(key string, val interface{})) {
	// attach cache for verifying values
	paramC := make(chan util.Param)
	lp.uiChan = paramC

	cache := util.NewCache()
	go cache.Run(paramC)

	expect := func(key string, val interface{}) {
		p := cache.Get(key)
		t.Logf("%s: %.f", key, p.Val) // REMOVE
		if p.Val != val {
			t.Errorf("%s wanted: %.0f, got %v", key, val, p.Val)
		}
	}

	return cache, expect
}

func TestChargedEnergyAtDisconnect(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	handler := mock.NewMockHandler(ctrl)
	rater := mock.NewMockChargeRater(ctrl)

	lp := &LoadPoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		chargeMeter: &Null{}, // silence nil panics
		chargeRater: rater,
		chargeTimer: &Null{}, // silence nil panics
		HandlerConfig: HandlerConfig{
			MinCurrent: lpMinCurrent,
			MaxCurrent: lpMaxCurrent,
		},
		handler: handler,
		status:  api.StatusC,
	}

	lp.Mode = api.ModeNow
	handler.EXPECT().Prepare().Return()
	attachListeners(t, lp)

	// attach cache for verifying values
	_, expectCache := cacheExpecter(t, lp)

	// start charging at 0 kWh
	handler.EXPECT().TargetCurrent().Return(int64(6))
	rater.EXPECT().ChargedEnergy().Return(0.0, nil)
	handler.EXPECT().Status().Return(api.StatusC, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(-1)

	// at 1:00h charging at 5 kWh
	clock.Add(time.Hour)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	handler.EXPECT().Status().Return(api.StatusC, nil)
	handler.EXPECT().SyncEnabled().Return()
	// handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(-1)
	expectCache("chargedEnergy", 5000.0)

	// at 1:00h stop charging at 5 kWh
	clock.Add(time.Second)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	handler.EXPECT().Status().Return(api.StatusB, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(-1)
	expectCache("chargedEnergy", 5000.0)

	// at 1:00h restart charging at 5 kWh
	clock.Add(time.Second)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	rater.EXPECT().ChargedEnergy().Return(5.0, nil)
	handler.EXPECT().Status().Return(api.StatusC, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(-1)
	expectCache("chargedEnergy", 5000.0)

	// at 1:30h continue charging at 7.5 kWh
	clock.Add(30 * time.Minute)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	rater.EXPECT().ChargedEnergy().Return(7.5, nil)
	handler.EXPECT().Status().Return(api.StatusC, nil)
	handler.EXPECT().SyncEnabled().Return()
	// handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(-1)
	expectCache("chargedEnergy", 7500.0)

	// at 2:00h stop charging at 10 kWh
	clock.Add(30 * time.Minute)
	handler.EXPECT().TargetCurrent().Return(int64(16))
	rater.EXPECT().ChargedEnergy().Return(10.0, nil)
	handler.EXPECT().Status().Return(api.StatusB, nil)
	handler.EXPECT().SyncEnabled().Return()
	handler.EXPECT().TargetCurrent().Return(int64(0)) // once more for status changes
	handler.EXPECT().Ramp(int64(16), true).Return(nil)
	lp.Update(-1)
	expectCache("chargedEnergy", 10000.0)

	ctrl.Finish()
}

func TestTargetSoC(t *testing.T) {
	ctrl := gomock.NewController(t)
	vhc := mock.NewMockVehicle(ctrl)

	tc := []struct {
		vehicle api.Vehicle
		target  int
		soc     float64
		res     bool
	}{
		{nil, 0, 0, false},     // never reached without vehicle
		{nil, 0, 10, false},    // never reached without vehicle
		{nil, 80, 0, false},    // never reached without vehicle
		{nil, 80, 80, false},   // never reached without vehicle
		{nil, 80, 100, false},  // never reached without vehicle
		{vhc, 0, 0, false},     // target disabled
		{vhc, 0, 10, false},    // target disabled
		{vhc, 80, 0, false},    // target not reached
		{vhc, 80, 80, true},    // target reached
		{vhc, 80, 100, true},   // target reached
		{vhc, 100, 100, false}, // target reached, let ev control deactivation
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		lp := &LoadPoint{
			vehicle: tc.vehicle,
			SoC: SoCConfig{
				Target: tc.target,
			},
			socCharge: tc.soc,
		}

		if res := lp.targetSocReached(); tc.res != res {
			t.Errorf("expected %v, got %v", tc.res, res)
		}
	}
}

func TestMinSoC(t *testing.T) {
	ctrl := gomock.NewController(t)
	vhc := mock.NewMockVehicle(ctrl)

	tc := []struct {
		vehicle api.Vehicle
		min     int
		soc     float64
		res     bool
	}{
		{nil, 0, 0, false},    // never reached without vehicle
		{nil, 0, 10, false},   // never reached without vehicle
		{nil, 80, 0, false},   // never reached without vehicle
		{nil, 80, 80, false},  // never reached without vehicle
		{nil, 80, 100, false}, // never reached without vehicle
		{vhc, 0, 0, false},    // min disabled
		{vhc, 0, 10, false},   // min disabled
		{vhc, 80, 0, true},    // min not reached
		{vhc, 80, 80, false},  // min reached
		{vhc, 80, 100, false}, // min reached
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		lp := &LoadPoint{
			vehicle: tc.vehicle,
			SoC: SoCConfig{
				Min: tc.min,
			},
			socCharge: tc.soc,
		}

		if res := lp.minSocNotReached(); tc.res != res {
			t.Errorf("expected %v, got %v", tc.res, res)
		}
	}
}
