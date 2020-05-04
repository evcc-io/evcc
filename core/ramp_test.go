package core

import (
	"testing"

	"github.com/andig/evcc/api"
	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
)

type nilCharger struct{}

func (c *nilCharger) Status() (api.ChargeStatus, error) { return api.StatusA, nil }
func (c *nilCharger) Enabled() (bool, error)            { return false, nil }
func (c *nilCharger) Enable(enable bool) error          { return nil }
func (c *nilCharger) MaxCurrent(current int64) error    { return nil }

func TestEnable(t *testing.T) {
	const minA = 6

	tc := []struct {
		enabledI, enabledO             bool
		targetCurrentI, targetCurrentO int64
	}{
		// any test with current != 0 or min will fail
		{false, false, 0, 0},
		{false, true, 0, 0},
		{false, true, minA, minA},
		{true, false, minA, minA},
		{true, true, minA, minA},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)

		r := &Ramp{
			clock:         clock.NewMock(),
			enabled:       tc.enabledI,
			targetCurrent: tc.targetCurrentI,
			charger:       &nilCharger{},
			MinCurrent:    minA,
		}

		r.chargerEnable(tc.enabledO)

		if r.enabled != tc.enabledO {
			t.Errorf("enabled: expected %s, got %s", status[tc.enabledO], status[r.enabled])
		}
		if r.targetCurrent != tc.targetCurrentO {
			t.Errorf("targetCurrent: expected %d, got %d", tc.targetCurrentO, r.targetCurrent)
		}

		ctrl.Finish()
	}
}

func TestSetCurrent(t *testing.T) {
	const minA = 6

	tc := []struct {
		enabledI, enabledO             bool
		targetCurrentI, targetCurrentO int64
	}{
		// any test with current != 0 or min will fail
		{false, false, 0, 0},
		{false, true, 0, 0},
		{false, true, minA, minA},
		{true, false, minA, minA},
		{true, true, minA, minA},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)

		r := &Ramp{
			clock:         clock.NewMock(),
			enabled:       tc.enabledI,
			targetCurrent: tc.targetCurrentI,
			charger:       &nilCharger{},
			MinCurrent:    minA,
		}

		r.setTargetCurrent(tc.targetCurrentO)

		if r.enabled != tc.enabledO {
			t.Errorf("enabled: expected %s, got %s", status[tc.enabledO], status[r.enabled])
		}
		if r.targetCurrent != tc.targetCurrentO {
			t.Errorf("targetCurrent: expected %d, got %d", tc.targetCurrentO, r.targetCurrent)
		}

		ctrl.Finish()
	}
}

func TestRampOn(t *testing.T) {
}

func TestRampOff(t *testing.T) {
}

func TestRampUpDown(t *testing.T) {
}
