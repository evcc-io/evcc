package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"
	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
)

const (
	lpMinCurrent int64 = 6
	lpMaxCurrent int64 = 16
)

func TestNew(t *testing.T) {
	lp := NewLoadPoint()

	if lp.Mode != api.ModeOff {
		t.Errorf("Mode %v", lp.Mode)
	}
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
	if lp.enabled {
		t.Errorf("enabled %v", lp.enabled)
	}
	if lp.charging {
		t.Errorf("charging %v", lp.charging)
	}
	if lp.targetCurrent != 0 {
		t.Errorf("targetCurrent %v", lp.targetCurrent)
	}
}

func newLoadPoint(charger api.Charger, pv, gm, cm api.Meter) *LoadPoint {
	lp := NewLoadPoint()
	lp.clock = clock.NewMock()
	lp.clock.(*clock.Mock).Add(time.Hour)

	lp.charger = charger
	lp.pvMeter = pv
	lp.gridMeter = gm

	// prevent assigning a nil pointer sake of
	// https://groups.google.com/forum/#!topic/golang-nuts/wnH302gBa4I/discussion
	if !(cm == nil || reflect.ValueOf(cm).IsNil()) {
		lp.chargeMeter = cm
	}

	uiChan := make(chan util.Param)
	notificationChan := make(chan push.Event)

	lp.Prepare(uiChan, notificationChan)

	go func() {
		for {
			select {
			case <-uiChan:
			case <-notificationChan:
			}
		}
	}()

	return lp
}

func newEnvironment(t *testing.T, ctrl *gomock.Controller, pm, gm, cm api.Meter) (*LoadPoint, *mock.MockCharger) {
	wb := mock.NewMockCharger(ctrl)

	wb.EXPECT().Enabled().Return(true, nil) // initial alignment with wb
	wb.EXPECT().MaxCurrent(lpMinCurrent)    // initial alignment with wb

	lp := newLoadPoint(wb, pm, gm, cm)
	if !lp.enabled {
		t.Errorf("enabled %v", lp.enabled)
	}
	if lp.guardUpdated != lp.clock.Now() {
		t.Errorf("guardUpdated %v", lp.guardUpdated)
	}

	return lp, wb
}

func TestMeterConfigurations(t *testing.T) {
	tc := []struct {
		gm, cm, pm bool
	}{
		// {false, false, false}, // no meter
		// {false, true, false}, // cm only
		{false, false, true}, // pm only
		{true, false, false}, // gm only
		{true, true, false},  // gm + cm
		{true, false, true},  // gm + pm
		{true, true, true},   // gm + cm + pm
	}

	fg := provider.FloatGetter(func() (float64, error) {
		return 1, nil
	})

	for _, tc := range tc {
		t.Logf("gm: %+v  cm: %v  pm: %v", tc.gm, tc.cm, tc.pm)

		var gm, pm, cm api.Meter
		if tc.gm {
			gm = meter.NewConfigurable(fg)
		}
		if tc.cm {
			cm = meter.NewConfigurable(fg)
		}
		if tc.pm {
			pm = meter.NewConfigurable(fg)
		}

		ctrl := gomock.NewController(t)
		lp, wb := newEnvironment(t, ctrl, pm, gm, cm)
		wb.EXPECT().Status().Return(api.StatusA, nil) // disconnected
		wb.EXPECT().Enable(false)                     // "off" mode

		lp.update()
	}
}

func TestInitialUpdate(t *testing.T) {
	tc := []struct {
		status api.ChargeStatus
		mode   api.ChargeMode
	}{
		{status: api.StatusA, mode: api.ModeOff},
		{status: api.StatusA, mode: api.ModeNow},
		{status: api.StatusA, mode: api.ModeMinPV},
		{status: api.StatusA, mode: api.ModePV},

		{status: api.StatusB, mode: api.ModeOff},
		{status: api.StatusB, mode: api.ModeNow},
		{status: api.StatusB, mode: api.ModeMinPV},
		{status: api.StatusB, mode: api.ModePV},

		{status: api.StatusC, mode: api.ModeOff},
		{status: api.StatusC, mode: api.ModeNow},
		{status: api.StatusC, mode: api.ModeMinPV},
		{status: api.StatusC, mode: api.ModePV},
	}

	for _, tc := range tc {
		t.Logf("%+v\n", tc)

		ctrl := gomock.NewController(t)

		pm := mock.NewMockMeter(ctrl)
		gm := mock.NewMockMeter(ctrl)
		cm := mock.NewMockMeter(ctrl)
		// cm = nil

		lp, wb := newEnvironment(t, ctrl, pm, gm, cm)
		lp.Mode = tc.mode

		wb.EXPECT().Status().Return(tc.status, nil)

		// values are relevant for PV case
		minPower := float64(lpMinCurrent) * lp.Voltage
		pm.EXPECT().CurrentPower().Return(minPower, nil)
		gm.EXPECT().CurrentPower().Return(float64(0), nil)
		if cm != nil {
			cm.EXPECT().CurrentPower().Return(minPower, nil)
		}

		// disable if not connected
		if tc.mode == api.ModeOff {
			wb.EXPECT().Enable(false)
		}

		// power up if now
		if tc.status != api.StatusA && tc.mode == api.ModeNow {
			wb.EXPECT().MaxCurrent(lpMaxCurrent)
		}

		lp.update()

		// max current if connected & mode now
		if tc.status != api.StatusA && tc.mode == api.ModeNow {
			if lp.targetCurrent != lpMaxCurrent {
				t.Errorf("targetCurrent %v", lp.targetCurrent)
			}
		}

		// min current in first cycle
		if tc.mode != api.ModeNow {
			if lp.targetCurrent != lpMinCurrent {
				t.Errorf("targetCurrent %v", lp.targetCurrent)
			}
		}

		// status c means charging
		if lp.charging != (tc.status == api.StatusC) {
			t.Errorf("charging %v", lp.charging)
		}

		ctrl.Finish()
	}
}

func TestImmediateOnOff(t *testing.T) {
	tc := []struct {
		status api.ChargeStatus
		mode   api.ChargeMode
	}{
		{status: api.StatusC, mode: api.ModePV},
	}

	for _, tc := range tc {
		t.Logf("%+v\n", tc)

		ctrl := gomock.NewController(t)

		pm := mock.NewMockMeter(ctrl)
		gm := mock.NewMockMeter(ctrl)
		cm := mock.NewMockMeter(ctrl)
		// cm = nil

		lp, wb := newEnvironment(t, ctrl, pm, gm, cm)
		lp.Mode = tc.mode

		// -- round 1
		wb.EXPECT().Status().Return(tc.status, nil)

		// values are relevant for PV case
		minPower := float64(lpMinCurrent) * lp.Voltage * float64(lp.Phases)
		pm.EXPECT().CurrentPower().Return(minPower, nil)
		gm.EXPECT().CurrentPower().Return(0.0, nil)
		if cm != nil {
			cm.EXPECT().CurrentPower().Return(minPower, nil)
		}

		// disable if not connected
		if tc.status != api.StatusA && tc.mode == api.ModeOff {
			wb.EXPECT().Enable(false)
		}

		// power up if now
		if tc.status != api.StatusA && tc.mode == api.ModeNow {
			wb.EXPECT().MaxCurrent(lpMaxCurrent)
		}

		lp.update()

		// max current if connected & mode now
		if tc.status != api.StatusA && tc.mode == api.ModeNow {
			if lp.targetCurrent != lpMaxCurrent {
				t.Errorf("targetCurrent %v", lp.targetCurrent)
			}
		}

		// min current in first cycle
		if tc.mode != api.ModeNow {
			if lp.targetCurrent != lpMinCurrent {
				t.Errorf("targetCurrent %v", lp.targetCurrent)
			}
		}

		// status c means charging
		if lp.charging != (tc.status == api.StatusC) {
			t.Errorf("charging %v", lp.charging)
		}

		// -- round 2
		wb.EXPECT().Status().Return(tc.status, nil)

		pm.EXPECT().CurrentPower().Return(minPower, nil)
		gm.EXPECT().CurrentPower().Return(-2*minPower, nil)
		if cm != nil {
			cm.EXPECT().CurrentPower().Return(1.0, nil)
		}

		wb.EXPECT().MaxCurrent(2 * lpMinCurrent)

		lp.update()

		// -- round 3
		t.Logf("%+v - 3 (status: %v, enabled: %v, current %d)\n", tc, lp.status, lp.enabled, lp.targetCurrent)

		wb.EXPECT().Status().Return(tc.status, nil)

		pm.EXPECT().CurrentPower().Return(minPower, nil)
		gm.EXPECT().CurrentPower().Return(-2*minPower, nil)
		if cm != nil {
			cm.EXPECT().CurrentPower().Return(1.0, nil)
		}

		wb.EXPECT().MaxCurrent(lpMinCurrent)

		lp.SetMode(api.ModeOff)
		lp.update()

		ctrl.Finish()
	}
}

func TestConsumedPower(t *testing.T) {
	tc := []struct {
		grid, pv, battery, consumed float64
	}{
		{0, 0, 0, 0},    // silent night
		{1, 0, 0, 1},    // grid import
		{0, 1, 0, 1},    // pv sign ignored
		{0, -1, 0, 1},   // pv sign ignored
		{1, 1, 0, 2},    // grid import + pv, pv sign ignored
		{1, -1, 0, 2},   // grid import + pv, pv sign ignored
		{0, 0, 1, 1},    // battery discharging
		{0, 0, -1, -1},  // battery charging -> negative result cannot occur in reality
		{1, -3, 1, 5},   // grid import + pv + battery discharging
		{1, -3, -1, 3},  // grid import + pv + battery charging -> should not happen in reality
		{0, -3, -1, 2},  // pv + battery charging
		{-1, -4, -1, 2}, // grid export + pv + battery charging
		{-1, -4, 0, 3},  // grid export + pv
	}

	for _, tc := range tc {
		res := consumedPower(tc.pv, tc.battery, tc.grid)
		if res != tc.consumed {
			t.Errorf("consumedPower wanted %.f, got %.f", tc.consumed, res)
		}
	}
}

func TestPVHysteresis(t *testing.T) {
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
		lp := LoadPoint{
			clock: clck,
			ChargerHandler: ChargerHandler{
				MinCurrent: lpMinCurrent,
				MaxCurrent: lpMaxCurrent,
				enabled:    tc.enabled,
			},
			Config: Config{
				Voltage: 100,
				Phases:  10,
				Enable: ThresholdConfig{
					Threshold: tc.enable,
					Delay:     dt,
				},
				Disable: ThresholdConfig{
					Threshold: tc.disable,
					Delay:     dt,
				},
			},
			gridPower: 0,
		}

		start := clck.Now()

		for step, se := range tc.series {
			clck.Set(start.Add(se.delay))
			lp.gridPower = se.site
			current := lp.maxCurrent(api.ModePV)

			if current != se.current {
				t.Errorf("step %d: wanted %d, got %d", step, se.current, current)
			}
		}
	}
}
