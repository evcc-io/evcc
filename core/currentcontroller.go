package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
)

type CurrentController struct {
	lp *Loadpoint

	// controller state, guarded by the loadpoint's mutex
	enabled        bool      // Charger enabled state
	phases         int       // Charger enabled phases
	offeredCurrent float64   // Charger current limit
	phaseTimer     time.Time // 1p3p switch timer
}

func newCurrentController(lp *Loadpoint) *CurrentController {
	return &CurrentController{lp: lp}
}

// ctrl returns the loadpoint's current controller or nil if the charger is natively power-controlled
func (lp *Loadpoint) ctrl() *CurrentController {
	ctrl, _ := lp.chargeController.(*CurrentController)
	return ctrl
}

func (c *CurrentController) setAndPublishEnabled(enabled bool) {
	if enabled != c.enabled {
		c.lp.log.DEBUG.Printf("charger %s", status[enabled])
		c.enabled = enabled
	}
	c.lp.publish(keys.Enabled, enabled)
}

// SetPower sets the charger to the given power target (0 disables).
// A positive target is clamped to the effective limits: a positive setpoint
// expresses that charging shall happen, hence a target below the feasible
// minimum charges at minimum power.
func (c *CurrentController) SetPower(power float64) error {
	// fixed phase configuration must match active phases before setting current
	if c.lp.connected() && c.scalePhasesRequired() {
		err := c.enforcePhases()
		if errors.Is(err, api.ErrNotAvailable) {
			// the charger cannot switch phases right now (e.g. EEBus charger
			// with an ISO 15118 vehicle). Adopt the configured phase count so
			// the switch is not re-attempted on every cycle (issue #29974).
			c.lp.SetPhases(c.lp.phasesConfigured)
			err = nil
		}
		return err
	}

	// surplus tracking: reconcile phases for the current surplus
	// TODO pass surplus explicitly once the controller owns its state
	surplusTracking := c.lp.surplus != nil
	if surplusTracking {
		surplus := *c.lp.surplus
		c.lp.surplus = nil

		if c.lp.hasPhaseSwitching() && c.lp.phaseSwitchCompleted() {
			c.pvScalePhases(surplus, c.effectiveMinCurrent(), c.effectiveMaxCurrent())
		}
	}

	if power <= 0 {
		return c.setLimit(0)
	}

	// full envelope requested: scale up phases if possible
	if power >= c.effectiveMaxPower() {
		return c.fastCharging()
	}

	// bottom envelope requested: scale down phases if possible. surplus-tracking
	// targets are excluded as their phase scaling is subject to hysteresis above.
	if !surplusTracking && power <= c.reachableMinPower() {
		return c.minCharging()
	}

	current := powerToCurrent(power, c.lp.ActivePhases())
	current = min(max(current, c.effectiveMinCurrent()), c.effectiveMaxCurrent())

	return c.setLimit(current)
}

// roundedCurrent rounds current down to full amps if charger or vehicle require it
func (c *CurrentController) roundedCurrent(current float64) float64 {
	// full amps only?
	if c.lp.coarseCurrent() {
		current = math.Trunc(current)
	}
	return current
}

// actualMaxChargeCurrent returns the maximum of all phase currents.
// If currents not measured falls back to offered current.
func (c *CurrentController) actualMaxChargeCurrent() float64 {
	if c.lp.chargeCurrents != nil {
		return max(c.lp.chargeCurrents[0], c.lp.chargeCurrents[1], c.lp.chargeCurrents[2])
	}
	if c.lp.charging() {
		return c.offeredCurrent
	}
	return 0
}

func (c *CurrentController) setMinCurrent() error {
	return c.setLimit(c.effectiveMinCurrent())
}

// setLimit applies charger current limits and enables/disables accordingly
func (c *CurrentController) setLimit(current float64) error {
	current = c.roundedCurrent(current)

	// apply circuit limits
	if c.lp.circuit != nil {
		currentLimit := c.lp.circuit.ValidateCurrent(c.actualMaxChargeCurrent(), current)

		activePhases := c.lp.ActivePhases()
		powerLimit := c.lp.circuit.ValidatePower(c.lp.chargePower, currentToPower(current, activePhases))
		currentLimitViaPower := powerToCurrent(powerLimit, activePhases)

		current = c.roundedCurrent(min(currentLimit, currentLimitViaPower))
	}

	// https://github.com/evcc-io/evcc/issues/16309
	effMinCurrent := c.effectiveMinCurrent()
	if effMaxCurrent := c.effectiveMaxCurrent(); effMinCurrent > effMaxCurrent {
		return fmt.Errorf("invalid config: min current %.3gA exceeds max current %.3gA", effMinCurrent, effMaxCurrent)
	}

	// set current
	if current != c.offeredCurrent && current >= effMinCurrent {
		var err error
		if charger, ok := api.Cap[api.ChargerEx](c.lp.charger); ok {
			err = charger.MaxCurrentMillis(current)
		} else {
			var ctrl api.CurrentController
			if cc, ok := api.Cap[api.CurrentController](c.lp.charger); ok {
				ctrl = cc
			} else if rv := reflect.Indirect(reflect.ValueOf(c.lp.charger)); rv.IsValid() && rv.Kind() == reflect.Struct {
				for i := range rv.NumField() {
					if field := rv.Field(i); field.CanInterface() {
						if cc, ok := api.Cap[api.CurrentController](field.Interface()); ok {
							ctrl = cc
							break
						}
					}
				}
			}

			if ctrl != nil {
				err = ctrl.MaxCurrent(int64(current))
			} else {
				err = api.ErrNotAvailable
			}
		}

		if err != nil {
			v := c.lp.GetVehicle()
			if vv, ok := api.Cap[api.Resurrector](v); ok && errors.Is(err, api.ErrAsleep) && !hasFeature(v, api.WakeUpDisabled) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				c.lp.log.DEBUG.Printf("set charge current limit: waking up vehicle")
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("set charge current limit %.3gA: %w", current, err)
		}

		c.lp.log.DEBUG.Printf("set charge current limit: %.3gA", current)
		c.offeredCurrent = current
		c.lp.bus.Publish(evChargeCurrent, current)
	}

	// set enabled/disabled
	if enabled := current >= effMinCurrent; enabled != c.enabled {
		if err := c.lp.charger.Enable(enabled); err != nil {
			v := c.lp.GetVehicle()
			if vv, ok := api.Cap[api.Resurrector](v); enabled && ok && errors.Is(err, api.ErrAsleep) && !hasFeature(v, api.WakeUpDisabled) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				c.lp.log.DEBUG.Printf("charger %s: waking up vehicle", status[enabled])
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("charger %s: %w", status[enabled], err)
		}

		c.setAndPublishEnabled(enabled)
		c.lp.chargerSwitched = c.lp.clock.Now()

		// ensure we always re-set current when enabling charger
		if !enabled {
			c.offeredCurrent = 0
		}

		c.lp.bus.Publish(evChargeCurrent, current)

		// start/stop vehicle wake-up timer
		if enabled {
			c.lp.startWakeUpTimer()
		} else {
			c.lp.stopWakeUpTimer()
		}
	}

	return nil
}

// syncCharger updates charger status and synchronizes it with expectations
func (c *CurrentController) syncCharger() error {
	enabled, err := c.lp.charger.Enabled()
	if err != nil {
		return fmt.Errorf("charger enabled: %w", err)
	}

	shouldBeConsistent := c.lp.shouldBeConsistent()

	if shouldBeConsistent {
		defer func() {
			c.setAndPublishEnabled(enabled)
		}()
	}

	// #1: check charger logic, fix charger state if necessary (for chargers that start charging while being disabled)
	if !enabled && c.lp.charging() {
		c.lp.log.WARN.Println("charger logic error: disabled but charging")

		// treat as enabled when charging for further validations
		enabled = true

		if shouldBeConsistent {
			if err := c.lp.charger.Enable(true); err != nil { // also enable charger to correct internal state
				return fmt.Errorf("charger enable: %w", err)
			}

			c.lp.elapsePVTimer() // elapse PV timer so loadpoint can immediately switch charger if necessary
			return nil
		}
	}

	// #2: sync charger
	switch {
	case enabled && c.enabled:
		// sync max current
		var (
			current float64
			err     error
		)

		// use chargers actual set current if available
		cg, isCg := api.Cap[api.CurrentGetter](c.lp.charger)
		if isCg {
			if current, err = cg.GetMaxCurrent(); err == nil {
				// smallest adjustment most PWM-Controllers can do is: 100%÷256×0,6A = 0.234A
				if delta := math.Abs(c.offeredCurrent - current); delta > 0.23 {
					if shouldBeConsistent && delta >= 1 {
						c.lp.log.WARN.Printf("charger logic error: current mismatch (got %.3gA, expected %.3gA) - make sure your interval is at least 30s", current, c.offeredCurrent)
					}
					c.offeredCurrent = current
					c.lp.bus.Publish(evChargeCurrent, c.offeredCurrent)
				}
			} else if !loadpoint.AcceptableError(err) {
				return fmt.Errorf("charger get max current: %w", err)
			}
		}

		// use measured phase currents as fallback if charger does not provide max current or does not currently relay from vehicle (TWC3)
		if !isCg || errors.Is(err, api.ErrNotAvailable) {
			// validate if current too high by more than 1A (https://github.com/evcc-io/evcc/issues/14731)
			if current := c.lp.GetMaxPhaseCurrent(); current > c.offeredCurrent+1.0 {
				if shouldBeConsistent && !c.lp.chargerHasFeature(api.Heating) {
					c.lp.log.WARN.Printf("charger logic error: current mismatch (got %.3gA measured, expected %.3gA) - make sure your interval is at least 30s", current, c.offeredCurrent)
				}
				c.offeredCurrent = current
				c.lp.bus.Publish(evChargeCurrent, c.offeredCurrent)
			}
		}

		// sync phases
		if shouldBeConsistent {
			if err := c.lp.syncChargerPhases(); err != nil {
				return err
			}
		}

	case enabled == c.enabled:
		// sync disabled state

	case !enabled && !c.lp.phaseSwitchCompleted():
		// some chargers (i.E. Easee in some configurations) disable themselves to be able to switch phases
		// -> enable charger
		if err := c.lp.charger.Enable(true); err != nil {
			return fmt.Errorf("charger enable: %w", err)
		}

	case shouldBeConsistent && (enabled || c.lp.connected()):
		// ignore disabled state if vehicle was disconnected (!c.enabled && !c.lp.connected)
		c.lp.log.WARN.Printf("charger out of sync: expected %vd, got %vd", status[c.enabled], status[enabled])
	}

	return nil
}
