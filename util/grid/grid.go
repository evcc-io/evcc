// Package grid converts between phase currents and power for the configured grid.
package grid

import (
	"math"
	"sync/atomic"
)

// Voltage is the nominal phase voltage, line-to-line for IT grids
const Voltage = 230.0

// minActiveCurrent is the current below which a line conductor is treated as idle
const minActiveCurrent = 1.0

// Grid connection values: three-phase four-wire (with neutral) or three-wire (without neutral)
const (
	Connection3P4W = "3p4w"
	Connection3P3W = "3p3w"
)

var (
	// noNeutral can change at runtime via the UI while the control loop reads it
	noNeutral atomic.Bool
)

// NoNeutral reports a three-wire grid without neutral (IT grid)
func NoNeutral() bool {
	return noNeutral.Load()
}

// SetNoNeutral sets whether the grid is a three-wire grid without neutral (IT grid)
func SetNoNeutral(v bool) {
	noNeutral.Store(v)
}

// Factor returns the multiplier converting per-phase current times Voltage into total power
func Factor(phases int) float64 {
	if NoNeutral() && phases == 3 {
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

// CurrentsToPower estimates total power from signed line currents
func CurrentsToPower(l1, l2, l3 float64) float64 {
	if !NoNeutral() {
		return Voltage * (l1 + l2 + l3)
	}

	// single phase loads draw identical current on two line conductors, the third one only shows measurement noise
	peak := maxAbs(l1, l2, l3)
	if idle := min(math.Abs(l1), math.Abs(l2), math.Abs(l3)); idle < minActiveCurrent && idle < math.Abs(peak)/2 {
		return Voltage * peak
	}

	return Voltage * (l1 + l2 + l3) / math.Sqrt(3)
}

// maxAbs returns the value with the largest magnitude, keeping its sign
func maxAbs(vals ...float64) float64 {
	var res float64
	for _, v := range vals {
		if math.Abs(v) > math.Abs(res) {
			res = v
		}
	}
	return res
}
