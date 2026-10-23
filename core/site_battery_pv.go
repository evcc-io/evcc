package core

import "math"

func finiteBatteryPVValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
