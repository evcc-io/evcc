package core

import (
	"math"

	"github.com/evcc-io/evcc/api"
)

// setConfiguredPhases sets the default phase configuration
func (lp *Loadpoint) setConfiguredPhases(phases int) {
	lp.Lock()
	defer lp.Unlock()

	lp.ConfiguredPhases = phases

	// publish 1p3p capability and phase configuration
	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		lp.publish(phasesConfigured, lp.ConfiguredPhases)
	} else {
		lp.publish(phasesConfigured, nil)
	}
}

// setPhases sets the number of enabled phases without modifying the charger
func (lp *Loadpoint) setPhases(phases int) {
	if lp.GetPhases() != phases {
		lp.Lock()
		lp.phases = phases
		lp.Unlock()

		// publish updated phase configuration
		lp.publish(phasesEnabled, lp.phases)

		// reset timer to disabled state
		lp.resetPhaseTimer()

		// measure phases after switching
		lp.resetMeasuredPhases()
	}
}

// resetMeasuredPhases resets measured phases to unknown on vehicle disconnect, phase switch or phase api call
func (lp *Loadpoint) resetMeasuredPhases() {
	lp.Lock()
	lp.measuredPhases = 0
	lp.Unlock()

	lp.publish(phasesActive, lp.activePhases())
}

// getMeasuredPhases provides synchronized access to measuredPhases
func (lp *Loadpoint) getMeasuredPhases() int {
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
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *Loadpoint) activePhases() int {
	physical := lp.GetPhases()
	vehicle := lp.getVehiclePhases()
	measured := lp.getMeasuredPhases()

	active := min(expect(vehicle), expect(physical), expect(measured))

	// sanity check - we should not assume less active phases than actually measured
	if measured > 0 && active < measured {
		lp.log.WARN.Printf("phase mismatch between %dp measured for %dp vehicle and %dp charger", measured, vehicle, physical)
	}

	return active
}

// maxActivePhases returns the maximum number of active phases for the meter.
func (lp *Loadpoint) maxActivePhases() int {
	physical := lp.GetPhases()
	measured := lp.getMeasuredPhases()
	vehicle := lp.getVehiclePhases()

	// during 1p or unknown config, 1p measured is not a restriction
	if physical <= 1 || vehicle == 1 {
		measured = 0
	}

	// if 1p3p supported then assume configured limit or 3p
	if _, ok := lp.charger.(api.PhaseSwitcher); ok {
		physical = lp.ConfiguredPhases
	}

	return min(expect(vehicle), expect(physical), expect(measured))
}

func (lp *Loadpoint) getVehiclePhases() int {
	if lp.vehicle != nil {
		return lp.vehicle.Phases()
	}

	return 0
}
