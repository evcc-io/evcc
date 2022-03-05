package core

import (
	"math"

	"github.com/evcc-io/evcc/api"
)

// setMeasuredPhases provides synchronized access to measuredPhases
func (lp *LoadPoint) setMeasuredPhases(phases int) {
	lp.Lock()
	lp.measuredPhases = phases
	lp.Unlock()

	// publish best guess if phases are reset
	if phases == 0 {
		phases = lp.activePhases(false)
	}

	lp.publish("activePhases", phases)
}

// getMeasuredPhases provides synchronized access to measuredPhases
func (lp *LoadPoint) getMeasuredPhases() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.measuredPhases
}

// assume 3p for switchable charger during startup
const unknownPhases = 3

func min(i ...int) int {
	v := math.MaxInt
	for _, i := range i {
		if i < v {
			v = i
		}
	}
	return v
}

func expect(phases int) int {
	if phases > 0 {
		return phases
	}
	return unknownPhases
}

// activePhases returns the number of expectedly active phases for the meter.
// If max is true, the maximum number of active phases is returned.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *LoadPoint) activePhases(max bool) int {
	physical := lp.GetPhases()
	measured := lp.getMeasuredPhases()
	vehicle := lp.getVehiclePhases()

	if max {
		// during 1p or unknown config, 1p measured is not a restriction
		if physical <= 1 || vehicle == 1 {
			measured = 0
		}

		// if 1p3p supported then assume 3p
		if _, ok := lp.charger.(api.ChargePhases); ok {
			physical = 3
		}
	}

	return min(expect(vehicle), expect(physical), expect(measured))
}

func (lp *LoadPoint) getVehiclePhases() int {
	if vehicle := lp.getVehicle(); vehicle != nil {
		return vehicle.Phases()
	}

	return 0
}
