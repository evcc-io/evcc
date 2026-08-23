package core

import (
	"github.com/evcc-io/evcc/core/keys"
)

// setPhasesConfigured sets the default phase configuration
func (lp *Loadpoint) setPhasesConfigured(phases int) {
	ctrl := lp.ctrl()
	ctrl.phasesConfigured = phases
	lp.publish(keys.PhasesConfigured, phases)
	lp.settings.SetInt(keys.PhasesConfigured, int64(phases))

	// configured phases are actual phases for non-1p3p charger
	// for 1p3p charger, configuration does not mean that the physical state has changed, so don't touch it
	if !lp.hasPhaseSwitching() {
		ctrl.setPhases(phases)
	}
}

// SetPhases sets the number of enabled phases without modifying the charger
func (lp *Loadpoint) SetPhases(phases int) {
	lp.Lock()
	defer lp.Unlock()
	lp.ctrl().setPhases(phases)
}

// ResetMeasuredPhases resets measured phases to unknown on vehicle disconnect, phase switch or phase api call
func (lp *Loadpoint) ResetMeasuredPhases() {
	lp.Lock()
	defer lp.Unlock()
	lp.ctrl().resetMeasuredPhases()
}

// GetMeasuredPhases provides synchronized access to measured phases
func (lp *Loadpoint) GetMeasuredPhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getMeasuredPhases()
}

// getMeasuredPhases provides access to measured phases
func (lp *Loadpoint) getMeasuredPhases() int {
	return lp.ctrl().measuredPhases
}

// ActivePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *Loadpoint) ActivePhases() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.activePhases()
}

// activePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *Loadpoint) activePhases() int {
	return lp.ctrl().activePhases()
}

// MinActivePhases returns the minimum number of active phases for the loadpoint.
func (lp *Loadpoint) MinActivePhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.minActivePhases()
}

// minActivePhases returns the minimum number of active phases for the loadpoint.
func (lp *Loadpoint) minActivePhases() int {
	return lp.ctrl().minActivePhases()
}

// MaxActivePhases returns the maximum number of active phases for the loadpoint.
func (lp *Loadpoint) MaxActivePhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.maxActivePhases()
}

// maxActivePhases returns the maximum number of active phases for the loadpoint.
func (lp *Loadpoint) maxActivePhases() int {
	return lp.ctrl().maxActivePhases()
}

func (lp *Loadpoint) getChargerPhysicalPhases() int {
	return lp.ctrl().getChargerPhysicalPhases()
}

func (lp *Loadpoint) hasPhaseSwitching() bool {
	return lp.ctrl().hasPhaseSwitching()
}

// phaseSwitchCompleted returns true if phase switch command should be already processed by the charger
func (lp *Loadpoint) phaseSwitchCompleted() bool {
	return lp.ctrl().phaseSwitchCompleted()
}
