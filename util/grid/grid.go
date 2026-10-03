// Package grid converts between phase currents and power for the configured grid.
package grid

import "math"

var (
	// Voltage is the nominal phase voltage, line-to-line for IT grids
	Voltage = 230.0

	// IT indicates a three-wire grid without neutral
	IT bool
)

// Factor returns the multiplier converting per-phase current times Voltage into total power
func Factor(phases int) float64 {
	if IT && phases == 3 {
		return math.Sqrt(3)
	}
	return float64(phases)
}

// CurrentToPower converts per-phase current to total power
func CurrentToPower(current float64, phases int) float64 {
	return current * Factor(phases) * Voltage
}

// PowerToCurrent converts total power to per-phase current
func PowerToCurrent(power float64, phases int) float64 {
	return power / (Factor(phases) * Voltage)
}

// CurrentsToPower estimates total power from line currents
func CurrentsToPower(l1, l2, l3 float64) float64 {
	if !IT {
		return Voltage * (l1 + l2 + l3)
	}

	// single phase loads draw identical current on two line conductors
	if min(l1, l2, l3) <= 0 {
		return Voltage * max(l1, l2, l3)
	}

	return Voltage * (l1 + l2 + l3) / math.Sqrt(3)
}
