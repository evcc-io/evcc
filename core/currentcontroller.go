package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/evcc-io/evcc/api"
)

type CurrentController struct {
	*Loadpoint
}

func newCurrentController(lp *Loadpoint) *CurrentController {
	return &CurrentController{Loadpoint: lp}
}

// MaxPower sets the charger to the given power target (0 disables)
func (lp *CurrentController) MaxPower(power float64) error {
	if power <= 0 {
		return lp.setLimit(0)
	}
	return lp.setLimit(powerToCurrent(power, lp.ActivePhases()))
}

// roundedCurrent rounds current down to full amps if charger or vehicle require it
func (lp *CurrentController) roundedCurrent(current float64) float64 {
	// full amps only?
	if lp.coarseCurrent() {
		current = math.Trunc(current)
	}
	return current
}

// setLimit applies charger current limits and enables/disables accordingly
func (lp *CurrentController) setLimit(current float64) error {
	current = lp.roundedCurrent(current)

	// apply circuit limits
	if lp.circuit != nil {
		var actualCurrent float64
		if lp.chargeCurrents != nil {
			actualCurrent = max(lp.chargeCurrents[0], lp.chargeCurrents[1], lp.chargeCurrents[2])
		} else if lp.charging() {
			actualCurrent = lp.offeredCurrent
		}

		currentLimit := lp.circuit.ValidateCurrent(actualCurrent, current)

		activePhases := lp.ActivePhases()
		powerLimit := lp.circuit.ValidatePower(lp.chargePower, currentToPower(current, activePhases))
		currentLimitViaPower := powerToCurrent(powerLimit, activePhases)

		current = lp.roundedCurrent(min(currentLimit, currentLimitViaPower))
	}

	// https://github.com/evcc-io/evcc/issues/16309
	effMinCurrent := lp.effectiveMinCurrent()
	if effMaxCurrent := lp.effectiveMaxCurrent(); effMinCurrent > effMaxCurrent {
		return fmt.Errorf("invalid config: min current %.3gA exceeds max current %.3gA", effMinCurrent, effMaxCurrent)
	}

	// set current
	if current != lp.offeredCurrent && current >= effMinCurrent {
		var err error
		if charger, ok := api.Cap[api.ChargerEx](lp.charger); ok {
			err = charger.MaxCurrentMillis(current)
		} else {
			var ctrl api.CurrentController
			if c, ok := api.Cap[api.CurrentController](lp.charger); ok {
				ctrl = c
			} else if rv := reflect.Indirect(reflect.ValueOf(lp.charger)); rv.IsValid() && rv.Kind() == reflect.Struct {
				for i := range rv.NumField() {
					if field := rv.Field(i); field.CanInterface() {
						if c, ok := api.Cap[api.CurrentController](field.Interface()); ok {
							ctrl = c
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
			v := lp.GetVehicle()
			if vv, ok := api.Cap[api.Resurrector](v); ok && errors.Is(err, api.ErrAsleep) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				lp.log.DEBUG.Printf("set charge current limit: waking up vehicle")
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("set charge current limit %.3gA: %w", current, err)
		}

		lp.log.DEBUG.Printf("set charge current limit: %.3gA", current)
		lp.offeredCurrent = current
		lp.bus.Publish(evChargeCurrent, current)
	}

	// set enabled/disabled
	if enabled := current >= effMinCurrent; enabled != lp.enabled {
		if err := lp.charger.Enable(enabled); err != nil {
			v := lp.GetVehicle()
			if vv, ok := api.Cap[api.Resurrector](v); enabled && ok && errors.Is(err, api.ErrAsleep) {
				// https://github.com/evcc-io/evcc/issues/8254
				// wakeup vehicle
				lp.log.DEBUG.Printf("charger %s: waking up vehicle", status[enabled])
				if err := vv.WakeUp(); err != nil {
					return fmt.Errorf("wake-up vehicle: %w", err)
				}
			}

			return fmt.Errorf("charger %s: %w", status[enabled], err)
		}

		lp.setAndPublishEnabled(enabled)
		lp.chargerSwitched = lp.clock.Now()

		// ensure we always re-set current when enabling charger
		if !enabled {
			lp.offeredCurrent = 0
		}

		lp.bus.Publish(evChargeCurrent, current)

		// start/stop vehicle wake-up timer
		if enabled {
			lp.startWakeUpTimer()
		} else {
			lp.stopWakeUpTimer()
		}
	}

	return nil
}
