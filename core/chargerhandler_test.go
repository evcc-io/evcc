package core

import (
	"testing"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/mock"
	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
)

const (
	minA        int64 = 6
	maxA        int64 = 16
	sensitivity       = 1
	dt                = time.Hour
)

func newChargerHandler(clock clock.Clock, mc api.Charger) ChargerHandler {
	r := NewChargerHandler("", clock, evbus.New())

	r.charger = mc
	r.Sensitivity = sensitivity
	r.guardUpdated = clock.Now()

	return r
}

func TestNewChargerHandler(t *testing.T) {
	r := NewChargerHandler("", nil, nil)

	if r.MinCurrent != minA {
		t.Errorf("expected %v, got %v", minA, r.MinCurrent)
	}
	if r.MaxCurrent != maxA {
		t.Errorf("expected %v, got %v", maxA, r.MaxCurrent)
	}
	if r.Sensitivity != 10 {
		t.Errorf("expected %v, got %v", 10, r.Sensitivity)
	}
	if r.GuardDuration != 5*time.Minute {
		t.Errorf("expected %v, got %v", 5*time.Minute, r.GuardDuration)
	}
}

func TestEnable(t *testing.T) {
	tc := []struct {
		enabledI       bool
		dt             time.Duration
		enable         bool
		targetCurrentI int64
		expect         func(*mock.MockCharger)
	}{
		// any test with current != 0 or min will fail
		{false, 0, false, 0, func(mc *mock.MockCharger) {
			// nop
		}},
		{false, 0, true, 0, func(mc *mock.MockCharger) {
			// nop
		}},
		{false, dt, true, 0, func(mc *mock.MockCharger) {
			mc.EXPECT().Enable(true).Return(nil)
		}},
		{false, 0, true, minA, func(mc *mock.MockCharger) {
			// nop
		}},
		{false, dt, true, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().Enable(true).Return(nil)
		}},
		{true, 0, false, minA, func(mc *mock.MockCharger) {
			// nop
		}},
		{true, dt, false, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().Enable(false).Return(nil)
		}},
		{true, 0, true, minA, func(mc *mock.MockCharger) {
			// nop
		}},
		{true, dt, true, minA, func(mc *mock.MockCharger) {
			// nop
		}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		t.Log(tc)

		clock := clock.NewMock()
		r := newChargerHandler(clock, mc)
		r.enabled = tc.enabledI
		r.targetCurrent = tc.targetCurrentI

		tc.expect(mc)
		clock.Add(tc.dt)

		if err := r.chargerEnable(tc.enable); err != nil {
			t.Error(err)
		}

		ctrl.Finish()
	}
}

func TestSetCurrent(t *testing.T) {
	tc := []struct {
		targetCurrentI, targetCurrent, targetCurrentO int64
		expect                                        func(*mock.MockCharger)
	}{
		{0, 0, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
		}},
		{minA, minA, minA, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
		}},
		{minA, 0, minA, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
		}},
		{minA, maxA, maxA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(maxA).Return(nil)
		}},
		{maxA, maxA, maxA, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
		}},
		{minA, 2 * maxA, maxA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(maxA).Return(nil)
		}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		t.Log(tc)

		clock := clock.NewMock()
		r := newChargerHandler(clock, mc)
		r.targetCurrent = tc.targetCurrentI

		tc.expect(mc)

		if err := r.setTargetCurrent(tc.targetCurrent); err != nil {
			t.Error(err)
		}

		if r.targetCurrent != tc.targetCurrentO {
			t.Errorf("targetCurrent: expected %d, got %d", tc.targetCurrentO, r.targetCurrent)
		}

		ctrl.Finish()
	}
}

func TestRampOn(t *testing.T) {
	tc := []struct {
		enabledI                      bool
		targetCurrentI, targetCurrent int64
		dt                            time.Duration
		expect                        func(*mock.MockCharger)
	}{
		// off at zero: set min
		{false, 0, minA, 0, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			// guard duration
		}},
		{false, 0, minA, dt, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			mc.EXPECT().Enable(true).Return(nil)
		}},
		// off at max: set min
		{false, maxA, minA, 0, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			// guard duration
		}},
		{false, maxA, minA, dt, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			mc.EXPECT().Enable(true).Return(nil)
		}},
		// off at min: set on
		{false, minA, minA, 0, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			// guard duration
		}},
		{false, minA, minA, dt, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			mc.EXPECT().Enable(true).Return(nil)
		}},
		// on at min, set min: set min
		{true, minA, minA, 0, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			// we are enabled: enable call omitted
		}},
		// on at max, set min: set min
		{true, maxA, minA, 0, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			// we are enabled: enable call omitted
		}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		t.Log(tc)

		clock := clock.NewMock()
		r := newChargerHandler(clock, mc)
		r.enabled = tc.enabledI
		r.targetCurrent = tc.targetCurrentI

		tc.expect(mc)
		clock.Add(tc.dt)

		if err := r.rampOn(tc.targetCurrent); err != nil {
			t.Error(err)
		}

		ctrl.Finish()
	}
}

func TestRampOff(t *testing.T) {
	tc := []struct {
		enabledI       bool
		targetCurrentI int64
		dt             time.Duration
		expect         func(*mock.MockCharger)
	}{
		// off at zero
		{false, 0, 0, func(mc *mock.MockCharger) {
			// we are off: enable call omitted
		}},
		// off at min
		{false, minA, 0, func(mc *mock.MockCharger) {
			// we are off: enable call omitted
		}},
		// off at max
		{false, maxA, 0, func(mc *mock.MockCharger) {
			// we are off: enable call omitted
		}},
		// on at min, disable
		{true, minA, 0, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			// guard duration
		}},
		{true, minA, dt, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			mc.EXPECT().Enable(false).Return(nil)
		}},
		// on at max, set min
		{true, maxA, 0, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			// we are not at min: enable call omitted
		}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		t.Log(tc)

		clock := clock.NewMock()
		r := newChargerHandler(clock, mc)
		r.enabled = tc.enabledI
		r.targetCurrent = tc.targetCurrentI

		tc.expect(mc)
		clock.Add(tc.dt)

		if err := r.rampOff(); err != nil {
			t.Error(err)
		}

		ctrl.Finish()
	}
}

func TestRampUpDown(t *testing.T) {
	tc := []struct {
		targetCurrentI, targetCurrent int64
		expect                        func(*mock.MockCharger)
	}{
		// no change at 0: nop
		{0, 0, func(mc *mock.MockCharger) {
			// nop
		}},
		// no change at min: nop
		{minA, minA, func(mc *mock.MockCharger) {
			// nop
		}},
		// at min: set <min
		{minA, minA - 100, func(mc *mock.MockCharger) {
			// nop
		}},
		// at min: set max
		{minA, maxA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA + sensitivity).Return(nil)
		}},
		// at max: set min
		{maxA, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(maxA - sensitivity).Return(nil)
		}},
		// at max: set >max
		{maxA, maxA + 100, func(mc *mock.MockCharger) {
			// nop
		}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		t.Log(tc)

		clock := clock.NewMock()
		r := newChargerHandler(clock, mc)
		r.enabled = true
		r.targetCurrent = tc.targetCurrentI

		tc.expect(mc)

		if err := r.rampUpDown(tc.targetCurrent); err != nil {
			t.Error(err)
		}

		ctrl.Finish()
	}
}
