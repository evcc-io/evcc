package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/api"
)

type batteryModeControllerTestImpl struct {
	modes []api.BatteryMode
}

func (m *batteryModeControllerTestImpl) BatteryModes() []api.BatteryMode {
	return m.modes
}

func (*batteryModeControllerTestImpl) SetBatteryMode(api.BatteryMode) error {
	return nil
}

type batteryModePowerControllerTestImpl struct{}

func (*batteryModePowerControllerTestImpl) SetBatteryModePower(api.BatteryMode, float64) error {
	return nil
}

type batteryModeOnlyTestMeter struct {
	api.Meter
	api.BatteryController
}

type batteryModePowerTestMeter struct {
	batteryModeOnlyTestMeter
	api.BatteryModePowerController
}

func TestClearMasterOfUncontrollableBatterySlave(t *testing.T) {
	allModes := []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge, api.BatteryDischarge}
	chargeOnly := []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge}

	tests := []struct {
		name         string
		modes        []api.BatteryMode
		powerSetter  bool
		noController bool
		master       any
		wantCleared  bool
	}{
		{name: "supported", modes: allModes, powerSetter: true, master: "master"},
		{name: "missing discharge mode", modes: chargeOnly, powerSetter: true, master: "master", wantCleared: true},
		{name: "missing power setter", modes: allModes, master: "master", wantCleared: true},
		{name: "no mode controller", noController: true, master: "master", wantCleared: true},
		{name: "empty master", master: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var meter api.Meter
			if test.noController {
				meter = &struct{ api.Meter }{}
			} else {
				modeController := &batteryModeControllerTestImpl{modes: test.modes}
				modeOnly := batteryModeOnlyTestMeter{BatteryController: modeController}
				if test.powerSetter {
					meter = &batteryModePowerTestMeter{
						batteryModeOnlyTestMeter:   modeOnly,
						BatteryModePowerController: &batteryModePowerControllerTestImpl{},
					}
				} else {
					meter = &modeOnly
				}
			}

			values := map[string]any{"master": test.master}
			warning := clearMasterOfUncontrollableBatterySlave("slave", values, meter)
			if (warning != nil) != test.wantCleared {
				t.Fatalf("warning presence: got %t, want %t", warning != nil, test.wantCleared)
			}
			if test.wantCleared && values["master"] != nil {
				t.Fatalf("master was not cleared: got %v", values["master"])
			}
			if test.wantCleared && (warning.Device != "slave" || warning.Master != test.master || warning.Reason != batteryConfigWarningUnsupportedSlave) {
				t.Fatalf("unexpected warning: %+v", warning)
			}
			if !test.wantCleared && values["master"] != test.master {
				t.Fatalf("master changed: got %v, want %v", values["master"], test.master)
			}
		})
	}
}
