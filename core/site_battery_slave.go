package core

import (
	"slices"

	"github.com/evcc-io/evcc/api"
)

// SupportsBatterySlaveControl reports whether a meter supports the modes and setpoint control required for battery slave control.
func SupportsBatterySlaveControl(meter api.Meter) bool {
	controller, ok := api.Cap[api.BatteryController](meter)
	if !ok {
		return false
	}

	modes := controller.BatteryModes()
	return slices.Contains(modes, api.BatteryCharge) &&
		slices.Contains(modes, api.BatteryDischarge) &&
		api.HasCap[api.BatteryModePowerController](meter)
}
