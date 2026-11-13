package core

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/grid"
)

// parsePhaseRotation converts the configured grid phase of charger L1 into 1..3, or 0 if not configured
func parsePhaseRotation(s string) (int, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "":
		return 0, nil
	case "L1":
		return 1, nil
	case "L2":
		return 2, nil
	case "L3":
		return 3, nil
	default:
		return 0, fmt.Errorf("invalid phase rotation: %s, must be L1, L2 or L3", s)
	}
}

// gridPhase maps a charger phase index 0..2 to its grid phase index using the cyclic rotation
func (lp *Loadpoint) gridPhase(chargerPhase int) int {
	return (lp.phaseRotation - 1 + chargerPhase) % 3
}

// gridPhaseMapping returns true if the configured grid phases are used for load management.
// That needs measured charge currents to see which phases are actually charging, and a grid
// with neutral: in IT grids single phase charging draws current on two line conductors.
func (lp *Loadpoint) gridPhaseMapping() bool {
	if lp.phaseRotation == 0 {
		return false
	}

	measured := lp.chargeMeter != nil && api.HasCap[api.PhaseCurrents](lp.chargeMeter)
	if measured && !grid.NoNeutral() {
		return true
	}

	lp.phaseRotationIgnored.Do(func() {
		reason := "charger or charge meter does not measure phase currents"
		if measured {
			reason = "grid without neutral (IT)"
		}
		lp.log.WARN.Printf("phase connection ignored, using all grid phases: %s", reason)
	})

	return false
}

// gridPhases returns the grid phases used when charging on the given number of phases.
// Without an active phase mapping any grid phase may be affected.
func (lp *Loadpoint) gridPhases(phases int) api.GridPhases {
	if !lp.gridPhaseMapping() || phases <= 0 || phases >= 3 {
		return api.AllGridPhases
	}

	var res api.GridPhases
	for i := range phases {
		res[lp.gridPhase(i)] = true
	}

	return res
}

// circuitPhases returns the grid phases a current change of the loadpoint applies to
func (lp *Loadpoint) circuitPhases() api.GridPhases {
	// the number of phases is not reliable until switching has completed
	if !lp.phaseSwitchCompleted() {
		return api.AllGridPhases
	}
	return lp.gridPhases(lp.ActivePhases())
}

// mappedCurrents returns the measured charge currents per grid phase
func (lp *Loadpoint) mappedCurrents() [3]float64 {
	var res [3]float64
	for i, current := range lp.chargeCurrents {
		res[lp.gridPhase(i)] = current
	}
	return res
}

// GetGridPhaseCurrents returns the charge currents per grid phase. Without an active
// mapping or measured currents the highest phase current is assumed on all grid phases.
func (lp *Loadpoint) GetGridPhaseCurrents() [3]float64 {
	lp.RLock()
	defer lp.RUnlock()

	if !lp.gridPhaseMapping() || lp.chargeCurrents == nil || lp.enabled && !lp.phaseSwitchCompleted() {
		current := lp.maxPhaseCurrent()
		return [3]float64{current, current, current}
	}

	return lp.mappedCurrents()
}

// MeasuredGridPhaseCurrents returns the measured charge currents per grid phase.
// ok is false without an active mapping, without measurement or while phases switch.
func (lp *Loadpoint) MeasuredGridPhaseCurrents() ([3]float64, bool) {
	lp.RLock()
	defer lp.RUnlock()

	if !lp.gridPhaseMapping() || lp.chargeCurrents == nil || !lp.phaseSwitchCompleted() {
		return [3]float64{}, false
	}

	return lp.mappedCurrents(), true
}

// circuitCurrents returns the current the loadpoint draws per grid phase for validating
// circuit limits. With an active mapping these are the measured currents, also while
// phases switch: crediting current to a phase it is not drawn from could overload it.
// Otherwise the highest phase current is assumed on all grid phases.
func (lp *Loadpoint) circuitCurrents() [3]float64 {
	if lp.gridPhaseMapping() && lp.chargeCurrents != nil {
		return lp.mappedCurrents()
	}

	current := lp.actualMaxChargeCurrent()
	return [3]float64{current, current, current}
}
