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
	if !ctrl.hasPhaseSwitching() {
		ctrl.setPhases(phases)
	}
}

// ActivePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *Loadpoint) ActivePhases() int {
	return lp.ctrl().ActivePhases()
}
