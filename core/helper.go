package core

import (
	"github.com/avast/retry-go/v3"
)

var (
	status   = map[bool]string{false: "disable", true: "enable"}
	presence = map[bool]string{false: "✗", true: "✓"}

	// retryOptions ist the default options set for retryable operations
	retryOptions = []retry.Option{retry.Attempts(3), retry.LastErrorOnly(true)}

	// Voltage global value
	Voltage float64
)

// powerToCurrent is a helper function to convert power to per-phase current
func powerToCurrent(power float64, phases int) float64 {
	if Voltage == 0 {
		panic("Voltage is not set")
	}
	return power / (float64(phases) * Voltage)
}

// consumedPower estimates how much power the charger might have consumed given it was the only load
// func consumedPower(pv, battery, grid float64) float64 {
// 	return math.Abs(pv) + battery + grid
// }

// sitePower returns the available delta power that the charger might additionally consume
// negative value: available power (grid export), positive value: grid import
func sitePower(grid, battery, residual float64) float64 {
	// For hybrid inverters, battery can be charged from DC power in excess of
	// inverter AC rating. This must be offset by the grid consumption when calculating
	// available site power.
	// (https://github.com/evcc-io/evcc/issues/2734)
	if grid > 0 && battery < 0 {
		battery += grid
	}
	return grid + battery + residual
}
