package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

// setPhasesConfigured sets the default phase configuration
func (lp *Loadpoint) setPhasesConfigured(phases int) {
	lp.phasesConfigured = phases
	lp.publish(keys.PhasesConfigured, lp.phasesConfigured)
	lp.settings.SetInt(keys.PhasesConfigured, int64(lp.phasesConfigured))

	// configured phases are actual phases for non-1p3p charger
	// for 1p3p charger, configuration does not mean that the physical state has changed, so don't touch it
	if !lp.hasPhaseSwitching() {
		lp.phases = phases
		lp.publish(keys.Phases, lp.phases)
	}
}

// setPhases sets the number of enabled phases without modifying the charger
func (lp *Loadpoint) setPhases(phases int) {
	if lp.GetPhases() != phases {
		lp.Lock()
		lp.phases = phases
		lp.Unlock()

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

	lp.publish(keys.PhasesActive, lp.ActivePhases())
}

// getMeasuredPhases provides synchronized access to measuredPhases
func (lp *Loadpoint) getMeasuredPhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.measuredPhases
}

// assume 3p for switchable charger during startup
const unknownPhases = 3

func expect(phases int) int {
	if phases > 0 {
		return phases
	}
	return unknownPhases
}

// ActivePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *Loadpoint) ActivePhases() int {
	physical := lp.getPhases()
	vehicle := lp.getVehiclePhases()
	measured := lp.getMeasuredPhases()
	charger := lp.getChargerPhysicalPhases()

	active := min(expect(vehicle), expect(physical), expect(measured), expect(charger))

	// sanity check - we should not assume less active phases than actually measured
	if measured > 0 && active < measured {
		lp.log.WARN.Printf("phase mismatch between %dp measured for %dp vehicle and %dp charger", measured, vehicle, physical)
	}

	return active
}

// minActivePhases returns the minimum number of active phases for the loadpoint.
func (lp *Loadpoint) minActivePhases() int {
	lp.RLock()
	phasesConfigured := lp.phasesConfigured
	lp.RUnlock()

	// 1p3p supported or limit 1p
	if lp.hasPhaseSwitching() || phasesConfigured == 1 {
		return 1
	}

	return lp.maxActivePhases()
}

// maxActivePhases returns the maximum number of active phases for the loadpoint.
func (lp *Loadpoint) maxActivePhases() int {
	physical := lp.getPhases()
	measured := lp.getMeasuredPhases()
	vehicle := lp.getVehiclePhases()
	charger := lp.getChargerPhysicalPhases()

	// during 1p or unknown config, 1p measured is not a restriction
	if physical <= 1 || vehicle == 1 || charger == 1 {
		measured = 0
	}

	// if 1p3p supported then assume configured limit or 3p
	if lp.hasPhaseSwitching() {
		lp.RLock()
		physical = lp.phasesConfigured
		lp.RUnlock()
	}

	return min(expect(vehicle), expect(physical), expect(measured), expect(charger))
}

func (lp *Loadpoint) getVehiclePhases() int {
	if vehicle := lp.GetVehicle(); vehicle != nil {
		return vehicle.Phases()
	}

	return 0
}

func (lp *Loadpoint) getChargerPhysicalPhases() int {
	if cc, ok := lp.charger.(api.PhaseDescriber); ok {
		return cc.Phases()
	}

	return 0
}

func (lp *Loadpoint) hasPhaseSwitching() bool {
	_, ok := lp.charger.(api.PhaseSwitcher)
	return ok
}
