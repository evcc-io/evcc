package implement

import "github.com/evcc-io/evcc/api"

// BatteryModes returns a static mode getter for use with BatteryController
func BatteryModes(modes ...api.BatteryMode) func() []api.BatteryMode {
	return func() []api.BatteryMode {
		return modes
	}
}
