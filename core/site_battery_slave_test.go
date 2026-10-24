package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

type batterySlaveTestController struct {
	modes []api.BatteryMode
}

func (c *batterySlaveTestController) BatteryModes() []api.BatteryMode {
	return c.modes
}

func (*batterySlaveTestController) SetBatteryMode(api.BatteryMode) error {
	return nil
}

type batterySlaveTestPowerController struct{}

func (*batterySlaveTestPowerController) SetBatteryModePower(api.BatteryMode, float64) error {
	return nil
}

func batterySlaveTestMeter(modes []api.BatteryMode, powerControl bool) api.Meter {
	controller := &batterySlaveTestController{modes: modes}
	if powerControl {
		return &struct {
			api.Meter
			api.BatteryController
			api.BatteryModePowerController
		}{
			BatteryController:          controller,
			BatteryModePowerController: &batterySlaveTestPowerController{},
		}
	}
	return &struct {
		api.Meter
		api.BatteryController
	}{BatteryController: controller}
}

func TestSupportsBatterySlaveControl(t *testing.T) {
	allModes := []api.BatteryMode{api.BatteryNormal, api.BatteryCharge, api.BatteryDischarge}
	chargeOnly := []api.BatteryMode{api.BatteryNormal, api.BatteryCharge}
	dischargeOnly := []api.BatteryMode{api.BatteryNormal, api.BatteryDischarge}

	tests := []struct {
		name  string
		meter api.Meter
		want  bool
	}{
		{name: "all required capabilities", meter: batterySlaveTestMeter(allModes, true), want: true},
		{name: "missing controller", meter: &struct{ api.Meter }{}},
		{name: "missing charge mode", meter: batterySlaveTestMeter(dischargeOnly, true)},
		{name: "missing discharge mode", meter: batterySlaveTestMeter(chargeOnly, true)},
		{name: "missing power setter", meter: batterySlaveTestMeter(allModes, false)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, SupportsBatterySlaveControl(test.meter))
		})
	}
}
