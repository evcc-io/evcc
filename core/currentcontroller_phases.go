package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

// assume 3p for switchable charger during startup
const unknownPhases = 3

func expect(phases int) int {
	if phases > 0 {
		return phases
	}
	return unknownPhases
}

// SetPhases sets the number of enabled phases without modifying the charger
func (c *CurrentController) SetPhases(phases int) {
	c.lp.Lock()
	defer c.lp.Unlock()
	c.setPhases(phases)
}

// setPhases sets the number of enabled phases without modifying the charger
func (c *CurrentController) setPhases(phases int) {
	if c.phases != phases {
		c.phases = phases

		// reset timer to disabled state
		c.resetPhaseTimer()

		// measure phases after switching
		c.resetMeasuredPhases()
	}
}

// resetPhaseTimer resets the phase switch timer to disabled state
func (c *CurrentController) resetPhaseTimer() {
	if c.phaseTimer.IsZero() {
		return
	}

	c.phaseTimer = time.Time{}
	c.lp.publishTimer(phaseTimer, 0, timerInactive)
}

// ResetMeasuredPhases resets measured phases to unknown on vehicle disconnect, phase switch or phase api call
func (c *CurrentController) ResetMeasuredPhases() {
	c.lp.Lock()
	defer c.lp.Unlock()
	c.resetMeasuredPhases()
}

// resetMeasuredPhases resets measured phases to unknown on vehicle disconnect, phase switch or phase api call
func (c *CurrentController) resetMeasuredPhases() {
	c.measuredPhases = 0
	c.lp.publish(keys.PhasesActive, c.activePhases())
}

// ActivePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (c *CurrentController) ActivePhases() int {
	c.lp.Lock()
	defer c.lp.Unlock()
	return c.activePhases()
}

// activePhases returns the number of expectedly active phases for the meter.
// If unknown for 1p3p chargers during startup it will assume 3p.
func (c *CurrentController) activePhases() int {
	physical := c.phases
	vehicle := c.getVehiclePhases()
	measured := c.measuredPhases
	charger := c.getChargerPhysicalPhases()

	active := min(expect(vehicle), expect(physical), expect(measured), expect(charger))

	// sanity check - we should not assume less active phases than actually measured
	if measured > 0 && active < measured {
		c.lp.log.WARN.Printf("phase mismatch between %dp measured for %dp vehicle and %dp charger", measured, vehicle, physical)
	}

	return active
}

// minActivePhases returns the minimum number of active phases for the loadpoint.
func (c *CurrentController) minActivePhases() int {
	if c.hasPhaseSwitching() || c.phasesConfigured == 1 {
		return 1
	}

	return c.maxActivePhases()
}

// MaxActivePhases returns the maximum number of active phases for the loadpoint.
func (c *CurrentController) MaxActivePhases() int {
	c.lp.RLock()
	defer c.lp.RUnlock()
	return c.maxActivePhases()
}

// maxActivePhases returns the maximum number of active phases for the loadpoint.
func (c *CurrentController) maxActivePhases() int {
	physical := c.phases
	measured := c.measuredPhases
	vehicle := c.getVehiclePhases()
	charger := c.getChargerPhysicalPhases()

	// during 1p or unknown config, 1p measured is not a restriction
	if physical <= 1 || vehicle == 1 || charger == 1 {
		measured = 0
	}

	// if 1p3p supported then assume configured limit or 3p
	if c.hasPhaseSwitching() {
		physical = c.phasesConfigured
	}

	return min(expect(vehicle), expect(physical), expect(measured), expect(charger))
}

func (c *CurrentController) getVehiclePhases() int {
	if v := c.lp.GetVehicle(); v != nil {
		return v.Phases()
	}

	return 0
}

func (c *CurrentController) getChargerPhysicalPhases() int {
	if cc, ok := api.Cap[api.PhaseDescriber](c.lp.charger); ok {
		return cc.Phases()
	}

	return 0
}

func (c *CurrentController) hasPhaseSwitching() bool {
	return api.HasCap[api.PhaseSwitcher](c.lp.charger)
}

// GetMeasuredPhases provides synchronized access to measured phases
func (c *CurrentController) GetMeasuredPhases() int {
	c.lp.RLock()
	defer c.lp.RUnlock()
	return c.measuredPhases
}

// phaseSwitchCompleted returns true if phase switch command should be already processed by the charger (so we can try to sync charger and loadpoint and are able to measure currents)
func (c *CurrentController) phaseSwitchCompleted() bool {
	return time.Since(c.phasesSwitched) > phaseSwitchDuration
}

// syncChargerPhases synchronizes the assumed phase state with the charger's actual state.
// Chargers may reconfigure phases internally, i.e. when the vehicle is (dis)connected.
func (c *CurrentController) syncChargerPhases() error {
	phases := c.lp.GetPhases()
	if !c.hasPhaseSwitching() || phases <= 0 {
		return nil
	}

	if pg, ok := api.Cap[api.PhaseGetter](c.lp.charger); ok {
		chargerPhases, err := pg.GetPhases()
		if err != nil {
			if errors.Is(err, api.ErrNotAvailable) {
				return nil
			}
			return fmt.Errorf("charger get phases: %w", err)
		}

		if chargerPhases > 0 && chargerPhases != phases {
			c.lp.log.WARN.Printf("charger logic error: phases mismatch (got %d, expected %d)", chargerPhases, phases)
			c.SetPhases(chargerPhases)
		}

		return nil
	}

	// use measured phase currents for active phases as fallback if charger does not provide phases
	chargerPhases := c.GetMeasuredPhases()
	if chargerPhases == 2 {
		chargerPhases = 3
	}

	if chargerPhases > phases {
		c.lp.log.WARN.Printf("charger logic error: phases mismatch (got %d measured, expected %d)", chargerPhases, phases)
		c.SetPhases(chargerPhases)
	}

	return nil
}
