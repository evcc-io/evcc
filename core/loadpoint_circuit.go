package core

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
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

// HasGridPhaseMapping returns true if the grid phases of the loadpoint are configured
func (lp *Loadpoint) HasGridPhaseMapping() bool {
	return lp.phaseRotation > 0
}

// gridPhases returns the grid phases used when charging on the given number of phases.
// Without a configured rotation any grid phase may be affected.
func (lp *Loadpoint) gridPhases(phases int) api.GridPhases {
	if lp.phaseRotation == 0 || phases <= 0 || phases >= 3 {
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

// GetGridPhaseCurrents returns the charge currents per grid phase. Without a configured
// rotation or measured currents the highest phase current is assumed on all grid phases.
func (lp *Loadpoint) GetGridPhaseCurrents() [3]float64 {
	lp.RLock()
	defer lp.RUnlock()

	if lp.phaseRotation == 0 || lp.chargeCurrents == nil || lp.enabled && !lp.phaseSwitchCompleted() {
		current := lp.maxPhaseCurrent()
		return [3]float64{current, current, current}
	}

	var res [3]float64
	for i, current := range lp.chargeCurrents {
		res[lp.gridPhase(i)] = current
	}

	return res
}
