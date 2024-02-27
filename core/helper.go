package core

import (
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
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = time.Second
	return bo
}

// powerToCurrent is a helper function to convert power to per-phase current
func powerToCurrent(power float64, phases int) float64 {
	if Voltage == 0 {
		panic("Voltage is not set")
	}
	return power / (float64(phases) * Voltage)
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
