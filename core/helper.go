package core

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util"
)

var (
	status   = map[bool]string{false: "disable", true: "enable"}
	presence = map[bool]string{false: "✗", true: "✓"}

	// Voltage global value
	Voltage float64
)

// bo returns an exponential backoff for reading meter power quickly
func bo() *backoff.ExponentialBackOff {
	return backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(time.Second))
}

// powerToCurrent is a helper function to convert power to per-phase current
func powerToCurrent(power float64, phases int) float64 {
	if Voltage == 0 {
		panic("Voltage is not set")
	}
	return power / (float64(phases) * Voltage)
}

// currentToPower is a helper function to convert current to sum power
func currentToPower(current float64, phases int) float64 {
	if Voltage == 0 {
		panic("Voltage is not set")
	}
	return current * float64(phases) * Voltage
}

// sitePower returns the available delta power that the charger might additionally consume
// negative value: available power (grid export), positive value: grid import
func sitePower(log *util.Logger, maxGrid, grid, battery, residual float64) float64 {
	// For hybrid inverters, battery can be charged from DC power in excess of
	// inverter AC rating. This battery charge must not be counted as available for AC consumption.
	// https://github.com/evcc-io/evcc/issues/2734, https://github.com/evcc-io/evcc/issues/2986
	if maxGrid > 0 && grid > maxGrid && battery < 0 {
		log.TRACE.Printf("ignoring excess DC charging due to grid consumption: %.0fW > %.0fW", grid, maxGrid)
		battery = 0
	}

	return grid + battery + residual
}

// printPtr returns a string representation of a pointer value
func printPtr[T any](format string, v *T) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf(format, *v)
}

func ptrValueEqual[T comparable](a, b *T) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	return a == nil && b == nil || (*a) == (*b)
}
