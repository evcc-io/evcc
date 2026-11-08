package core

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util/grid"
)

// setPhasesConfigured sets the default phase configuration
func (lp *Loadpoint) setPhasesConfigured(phases int) {
	lp.phasesConfigured = phases
	lp.publish(keys.PhasesConfigured, lp.phasesConfigured)
	lp.settings.SetInt(keys.PhasesConfigured, int64(lp.phasesConfigured))

	// configured phases are actual phases for non-1p3p charger
	// for 1p3p charger, configuration does not mean that the physical state has changed, so don't touch it
	if !lp.hasPhaseSwitching() {
		lp.setPhases(phases)
	}
}

// SetPhases sets the number of enabled phases without modifying the charger
func (lp *Loadpoint) SetPhases(phases int) {
	lp.Lock()
	defer lp.Unlock()
	lp.setPhases(phases)
}

// setPhases sets the number of enabled phases without modifying the charger
func (lp *Loadpoint) setPhases(phases int) {
	if lp.phases != phases {
		lp.phases = phases

		// reset timer to disabled state
		lp.resetPhaseTimer()

		// measure phases after switching
		lp.resetMeasuredPhases()
	}
}

// ResetMeasuredPhases resets measured phases to unknown on vehicle disconnect, phase switch or phase api call
func (lp *Loadpoint) ResetMeasuredPhases() {
	lp.Lock()
	defer lp.Unlock()
	lp.resetMeasuredPhases()
}

// resetMeasuredPhases resets measured phases to unknown on vehicle disconnect, phase switch or phase api call
func (lp *Loadpoint) resetMeasuredPhases() {
	lp.measuredPhases = 0
	lp.publish(keys.PhasesActive, lp.activePhases())
}

// GetMeasuredPhases provides synchronized access to measuredPhases
func (lp *Loadpoint) GetMeasuredPhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getMeasuredPhases()
}

// getMeasuredPhases provides synchronized access to measuredPhases
func (lp *Loadpoint) getMeasuredPhases() int {
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
	lp.Lock()
	defer lp.Unlock()
	return lp.activePhases()
}

// activePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (lp *Loadpoint) activePhases() int {
	physical := lp.phases
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

// MinActivePhases returns the minimum number of active phases for the loadpoint.
func (lp *Loadpoint) MinActivePhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.minActivePhases()
}

// minActivePhases returns the minimum number of active phases for the loadpoint.
func (lp *Loadpoint) minActivePhases() int {
	// configured phases are both minimum and maximum, only automatic scales down to 1p
	if lp.hasPhaseSwitching() && lp.phasesConfigured == 0 {
		return 1
	}

	return lp.maxActivePhases()
}

// MaxActivePhases returns the maximum number of active phases for the loadpoint.
func (lp *Loadpoint) MaxActivePhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.maxActivePhases()
}

// maxActivePhases returns the maximum number of active phases for the loadpoint.
func (lp *Loadpoint) maxActivePhases() int {
	physical := lp.phases
	measured := lp.getMeasuredPhases()
	vehicle := lp.getVehiclePhases()
	charger := lp.getChargerPhysicalPhases()

	// during 1p or unknown config, 1p measured is not a restriction
	if physical <= 1 || vehicle == 1 || charger == 1 {
		measured = 0
	}

	// if 1p3p supported then assume configured limit or 3p
	if lp.hasPhaseSwitching() {
		physical = lp.phasesConfigured
	}

	return min(expect(vehicle), expect(physical), expect(measured), expect(charger))
}

func (lp *Loadpoint) getVehiclePhases() int {
	if v := lp.GetVehicle(); v != nil {
		return v.Phases()
	}

	return 0
}

func (lp *Loadpoint) getChargerPhysicalPhases() int {
	if cc, ok := api.Cap[api.PhaseDescriber](lp.charger); ok {
		return cc.Phases()
	}

	return 0
}

func (lp *Loadpoint) hasPhaseSwitching() bool {
	return api.HasCap[api.PhaseSwitcher](lp.charger)
}

// syncChargerPhases synchronizes the assumed phase state with the charger's actual state.
// Chargers may reconfigure phases internally, i.e. when the vehicle is (dis)connected.
// Unknown phases (0, i.e. after startup) are seeded from the charger without warning.
func (lp *Loadpoint) syncChargerPhases(disabled bool) error {
	phases := lp.GetPhases()
	if !lp.hasPhaseSwitching() {
		return nil
	}

	if pg, ok := api.Cap[api.PhaseGetter](lp.charger); ok {
		chargerPhases, err := pg.GetPhases()
		if err != nil {
			if errors.Is(err, api.ErrNotAvailable) {
				return nil
			}
			return fmt.Errorf("charger get phases: %w", err)
		}

		if chargerPhases > 0 && chargerPhases != phases {
			// a disabled charger may switch phases without the loadpoint
			if phases > 0 && !disabled {
				lp.log.WARN.Printf("charger logic error: phases mismatch (got %d, expected %d)", chargerPhases, phases)
			}
			lp.SetPhases(chargerPhases)
		}

		return nil
	}

	// measured phases are stale while disabled, keep the enabled-only fallback
	if disabled {
		return nil
	}

	// use measured phase currents for active phases as fallback if charger does not provide phases
	chargerPhases := lp.GetMeasuredPhases()
	if chargerPhases == 2 {
		chargerPhases = 3
	}

	// 1p measured does not confirm 1p enabled (1p vehicle on 3p), hence never seeds unknown phases
	if chargerPhases > max(phases, 1) {
		if phases > 0 {
			lp.log.WARN.Printf("charger logic error: phases mismatch (got %d measured, expected %d)", chargerPhases, phases)
		}
		lp.SetPhases(chargerPhases)
	}

	return nil
}

// GetPhasesUpPower returns the 1p→3p up-switch threshold in W (0 = default).
func (lp *Loadpoint) GetPhasesUpPower() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.PhasesUpPower
}

// GetPhasesDownPower returns the 3p→1p down-switch threshold in W (0 = default).
func (lp *Loadpoint) GetPhasesDownPower() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.PhasesDownPower
}

// SetPhasesHysteresis sets the 1p/3p switching thresholds in W (0 = default).
// Explicit thresholds must form a hysteresis band (down below up) or the
// switching would oscillate. For unset values (0, 0) the derived defaults
// can overlap (e.g. 6-16 A: 16A x 230V < 6A x 3 x 230V); in that case the
// defaults are kept with a warning, consistent with the YAML config path.
func (lp *Loadpoint) SetPhasesHysteresis(up, down float64) error {
	if up < 0 || down < 0 {
		return fmt.Errorf("phase hysteresis: thresholds must not be negative (up: %.0f W, down: %.0f W)", up, down)
	}
	effUp := effectivePhasesUpThreshold(lp, up)
	effDown := effectivePhasesDownThreshold(lp, down)
	if effDown >= effUp {
		if up == 0 && down == 0 {
			lp.log.WARN.Printf("phase hysteresis: default down threshold (%.0fW) must be below up threshold (%.0fW), set explicit thresholds", effDown, effUp)
		} else {
			return fmt.Errorf("phase hysteresis: down threshold (%.0f W) must be below up threshold (%.0f W)", effDown, effUp)
		}
	}

	lp.log.DEBUG.Printf("set phase hysteresis: up above %.0f W, down below %.0f W (0 = default)", up, down)

	lp.Lock()
	lp.PhasesUpPower = up
	lp.PhasesDownPower = down
	lp.Unlock()

	lp.publish(keys.PhasesUpPower, up)
	lp.publish(keys.PhasesDownPower, down)
	lp.settings.SetFloat(keys.PhasesUpPower, up)
	lp.settings.SetFloat(keys.PhasesDownPower, down)

	return nil
}

// effectivePhasesUpThreshold returns the effective 1p→3p up threshold in W,
// applying the derived default for an unset value
func effectivePhasesUpThreshold(lp *Loadpoint, up float64) float64 {
	if up > 0 {
		return up
	}
	return grid.CurrentToPower(lp.GetMaxCurrent(), 1)
}

// effectivePhasesDownThreshold returns the effective 3p→1p down threshold in
// W, applying the derived default for an unset value (3 phases as worst case)
func effectivePhasesDownThreshold(lp *Loadpoint, down float64) float64 {
	if down > 0 {
		return down
	}
	return grid.CurrentToPower(lp.GetMinCurrent(), 3)
}

// GetPhasesMaxCurrent3 returns the 3-phase max charge current in A (0 = maxCurrent)
func (lp *Loadpoint) GetPhasesMaxCurrent3() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.PhasesMaxCurrent3
}

// SetPhasesMaxCurrent3 sets the 3-phase max charge current in A (0 = maxCurrent)
func (lp *Loadpoint) SetPhasesMaxCurrent3(current float64) error {
	if current < 0 {
		return fmt.Errorf("phasesmaxcurrent3: current must not be negative (%.1f A)", current)
	}

	lp.RLock()
	minCurrent := lp.minCurrent
	maxCurrent := lp.maxCurrent
	lp.RUnlock()

	if current > 0 {
		if current > maxCurrent {
			lp.log.WARN.Printf("phasesmaxcurrent3 (%.1fA) above maxCurrent (%.1fA)", current, maxCurrent)
		}
		if current < minCurrent {
			// a 3p cap below the min current would make setLimit fail while
			// charging on 3 phases, so clamp to the min current
			lp.log.WARN.Printf("phasesmaxcurrent3 (%.1fA) below minCurrent (%.1fA), using minCurrent", current, minCurrent)
			current = minCurrent
		}
	}

	lp.Lock()
	lp.PhasesMaxCurrent3 = current
	lp.Unlock()

	lp.log.DEBUG.Printf("set phases max current 3p: %.1fA (0 = default)", current)
	lp.publish(keys.PhasesMaxCurrent3, current)
	lp.settings.SetFloat(keys.PhasesMaxCurrent3, current)

	return nil
}
