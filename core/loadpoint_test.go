package core

import (
	"testing"
	"time"

	"github.com/andig/evcc/api"
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

func attachListeners(lp *LoadPoint) {
	uiChan := make(chan util.Param)
	pushChan := make(chan push.Event)

	go func() {
		for {
			select {
			case <-uiChan:
			case <-pushChan:
			}
		}
	}()

	lp.uiChan = uiChan
	lp.pushChan = pushChan
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
			h.EXPECT().Ramp(int64(0), true)
		}},
		{api.StatusA, api.ModeNow, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(lpMinCurrent, true)
		}},
		{api.StatusA, api.ModeMinPV, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(lpMinCurrent)
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
			h.EXPECT().Ramp(lpMinCurrent) // min since update called with 0
		}},
		{api.StatusB, api.ModePV, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0)) // zero since update called with 0
			// h.EXPECT().Enabled().Return(false) // short-circuited due to status != C
		}},

		{api.StatusC, api.ModeOff, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0), true)
		}},
		{api.StatusC, api.ModeNow, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(lpMaxCurrent, true)
		}},
		{api.StatusC, api.ModeMinPV, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(lpMinCurrent) // min since update called with 0
		}},
		{api.StatusC, api.ModePV, func(h *mock.MockHandler) {
			h.EXPECT().Ramp(int64(0)) // zero since update called with 0
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
			chargeMeter: &Null{}, //silence nil panics
			chargeRater: &Null{}, //silence nil panics
			chargeTimer: &Null{}, //silence nil panics
			HandlerConfig: HandlerConfig{
				MinCurrent: lpMinCurrent,
				MaxCurrent: lpMaxCurrent,
			},
			handler: handler,
			status:  tc.status, // no status change
		}

		attachListeners(lp)

		handler.EXPECT().Status().Return(tc.status, nil)
		handler.EXPECT().TargetCurrent().Return(int64(0))

		if tc.status != api.StatusA {
			handler.EXPECT().SyncEnabled()
		}

		if tc.mode == api.ModeMinPV || tc.mode == api.ModePV {
			handler.EXPECT().TargetCurrent().Return(int64(0))
		}

		if tc.expect != nil {
			tc.expect(handler)
		}

		lp.Mode = tc.mode
		lp.Update(0)

		ctrl.Finish()
	}
}

// func TestConsumedPower(t *testing.T) {
// 	tc := []struct {
// 		grid, pv, battery, consumed float64
// 	}{
// 		{0, 0, 0, 0},    // silent night
// 		{1, 0, 0, 1},    // grid import
// 		{0, 1, 0, 1},    // pv sign ignored
// 		{0, -1, 0, 1},   // pv sign ignored
// 		{1, 1, 0, 2},    // grid import + pv, pv sign ignored
// 		{1, -1, 0, 2},   // grid import + pv, pv sign ignored
// 		{0, 0, 1, 1},    // battery discharging
// 		{0, 0, -1, -1},  // battery charging -> negative result cannot occur in reality
// 		{1, -3, 1, 5},   // grid import + pv + battery discharging
// 		{1, -3, -1, 3},  // grid import + pv + battery charging -> should not happen in reality
// 		{0, -3, -1, 2},  // pv + battery charging
// 		{-1, -4, -1, 2}, // grid export + pv + battery charging
// 		{-1, -4, 0, 3},  // grid export + pv
// 	}

// 	for _, tc := range tc {
// 		res := consumedPower(tc.pv, tc.battery, tc.grid)
// 		if res != tc.consumed {
// 			t.Errorf("consumedPower wanted %.f, got %.f", tc.consumed, res)
// 		}
// 	}
// }

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

			current := lp.maxCurrent(api.ModePV, se.site)

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
	current := lp.maxCurrent(api.ModePV, sitePower)

	if current != 0 {
		t.Errorf("PV mode could not disable charger as expected. Expected 0, got %d", current)
	}

	ctrl.Finish()
}
