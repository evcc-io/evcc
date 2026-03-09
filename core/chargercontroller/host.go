package chargercontroller

import "github.com/evcc-io/evcc/api"

// Host provides the controller with access to vehicle-dependent state
// and loadpoint callbacks. The loadpoint implements this interface.
type Host interface {
	// GetVehicle returns the currently active vehicle, or nil.
	// The controller queries vehicle limits, phases, and features directly.
	GetVehicle() api.Vehicle

	// WakeUpVehicle attempts to wake a sleeping vehicle.
	// Returns nil if no vehicle is connected or it has no Resurrector.
	WakeUpVehicle() error

	// Charging returns true if the charger status is C (actively charging).
	Charging() bool

	// StartWakeUpTimer starts the vehicle wake-up timeout.
	StartWakeUpTimer()

	// StopWakeUpTimer stops the vehicle wake-up timeout.
	StopWakeUpTimer()
}
