package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/mock"
	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
)

const (
	minA int64 = 6
	maxA int64 = 16
)

type nilCharger struct{}

func (c *nilCharger) Status() (api.ChargeStatus, error) { return api.StatusA, nil }
func (c *nilCharger) Enabled() (bool, error)            { return false, nil }
func (c *nilCharger) Enable(enable bool) error          { return nil }
func (c *nilCharger) MaxCurrent(current int64) error    { return nil }

func TestEnable(t *testing.T) {
	tc := []struct {
		enabledI, enabledO             bool
		targetCurrentI, targetCurrentO int64
		expect                         func(*mock.MockCharger)
	}{
		// any test with current != 0 or min will fail
		{false, false, 0, 0, func(mc *mock.MockCharger) {}},
		{false, true, 0, 0, func(mc *mock.MockCharger) {
			mc.EXPECT().Enable(true).Return(nil)
		}},
		{false, true, minA, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().Enable(true).Return(nil)
		}},
		{true, false, minA, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().Enable(false).Return(nil)
		}},
		{true, true, minA, minA, func(mc *mock.MockCharger) {}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		r := &Ramp{
			clock:         clock.NewMock(),
			bus:           evbus.New(),
			enabled:       tc.enabledI,
			targetCurrent: tc.targetCurrentI,
			charger:       mc,
			MinCurrent:    minA,
		}

		tc.expect(mc)
		r.chargerEnable(tc.enabledO)

		if r.enabled != tc.enabledO {
			t.Errorf("enabled: expected %s, got %s", status[tc.enabledO], status[r.enabled])
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

		r := &Ramp{
			clock:         clock.NewMock(),
			bus:           evbus.New(),
			MinCurrent:    minA,
			MaxCurrent:    maxA,
			targetCurrent: tc.targetCurrentI,
			charger:       mc,
		}

		tc.expect(mc)
		r.setTargetCurrent(tc.targetCurrent)

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
		expect                        func(*mock.MockCharger)
	}{
		// off at zero: set min
		{false, 0, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			mc.EXPECT().Enable(true).Return(nil)
		}},
		// off at max: set min
		{false, maxA, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			mc.EXPECT().Enable(true).Return(nil)
		}},
		// off at min: set on
		{false, minA, minA, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			mc.EXPECT().Enable(true).Return(nil)
		}},
		// on at min, set min: set min
		{true, minA, minA, func(mc *mock.MockCharger) {
			// we are at min: current call omitted
			// we are enabled: enable call omitted
		}},
		// on at max, set min: set min
		{true, maxA, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			// we are enabled: enable call omitted
		}},
		// on at max, set min: set on
		{true, maxA, minA, func(mc *mock.MockCharger) {
			mc.EXPECT().MaxCurrent(minA).Return(nil)
			// we are enabled: enable call omitted
		}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		r := &Ramp{
			clock:         clock.NewMock(),
			bus:           evbus.New(),
			MinCurrent:    minA,
			MaxCurrent:    maxA,
			enabled:       tc.enabledI,
			targetCurrent: tc.targetCurrentI,
			charger:       mc,
		}

		tc.expect(mc)
		r.rampOn(tc.targetCurrent)

		ctrl.Finish()
	}
}

func TestRampOff(t *testing.T) {
	tc := []struct {
		enabledI       bool
		targetCurrentI int64
		expect         func(*mock.MockCharger)
	}{
		// off at zero: set min
		{false, 0, func(mc *mock.MockCharger) {
			// we are off: enable call omitted
		}},
		// off at max: set min
		{false, maxA, func(mc *mock.MockCharger) {
			// we are off: enable call omitted
		}},
		// // off at min: set on
		{false, minA, func(mc *mock.MockCharger) {
			// we are off: enable call omitted
		}},
		// // on at min, set min: set min
		// {true, minA, func(mc *mock.MockCharger) {
		// 	// we are at min: current call omitted
		// 	// we are enabled: enable call omitted
		// }},
		// // on at max, set min: set min
		// {true, maxA, func(mc *mock.MockCharger) {
		// 	mc.EXPECT().MaxCurrent(minA).Return(nil)
		// 	// we are enabled: enable call omitted
		// }},
		// // on at max, set min: set on
		// {true, maxA, func(mc *mock.MockCharger) {
		// 	mc.EXPECT().MaxCurrent(minA).Return(nil)
		// 	// we are enabled: enable call omitted
		// }},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		mc := mock.NewMockCharger(ctrl)

		r := &Ramp{
			clock:         clock.NewMock(),
			bus:           evbus.New(),
			MinCurrent:    minA,
			MaxCurrent:    maxA,
			enabled:       tc.enabledI,
			targetCurrent: tc.targetCurrentI,
			charger:       mc,
		}

		tc.expect(mc)
		r.rampOff()

		ctrl.Finish()
	}
}

func TestRampUpDown(t *testing.T) {
}
