package core

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
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
