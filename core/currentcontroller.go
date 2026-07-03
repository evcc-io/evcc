package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/evcc-io/evcc/api"
)

type CurrentController struct {
	lp *Loadpoint
}

func newCurrentController(lp *Loadpoint) *CurrentController {
	return &CurrentController{lp: lp}
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

func (c *CurrentController) setMinCurrent() error {
	return c.setLimit(c.effectiveMinCurrent())
}

// setLimit applies charger current limits and enables/disables accordingly
func (c *CurrentController) setLimit(current float64) error {
	current = c.roundedCurrent(current)

	// apply circuit limits
	if c.lp.circuit != nil {
		var actualCurrent float64
		if c.lp.chargeCurrents != nil {
			actualCurrent = max(c.lp.chargeCurrents[0], c.lp.chargeCurrents[1], c.lp.chargeCurrents[2])
		} else if c.lp.charging() {
			actualCurrent = c.lp.offeredCurrent
		}

		currentLimit := c.lp.circuit.ValidateCurrent(actualCurrent, current)

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
	if current != c.lp.offeredCurrent && current >= effMinCurrent {
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
			if vv, ok := api.Cap[api.Resurrector](v); ok && errors.Is(err, api.ErrAsleep) {
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
		c.lp.offeredCurrent = current
		c.lp.bus.Publish(evChargeCurrent, current)
	}

	// set enabled/disabled
	if enabled := current >= effMinCurrent; enabled != c.lp.enabled {
		if err := c.lp.charger.Enable(enabled); err != nil {
			v := c.lp.GetVehicle()
			if vv, ok := api.Cap[api.Resurrector](v); enabled && ok && errors.Is(err, api.ErrAsleep) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				c.lp.log.DEBUG.Printf("charger %s: waking up vehicle", status[enabled])
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("charger %s: %w", status[enabled], err)
		}

		c.lp.setAndPublishEnabled(enabled)
		c.lp.chargerSwitched = c.lp.clock.Now()

		// ensure we always re-set current when enabling charger
		if !enabled {
			c.lp.offeredCurrent = 0
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
