package core

import (
	"errors"
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

func (lp *Loadpoint) evaluateDehumidifierState(currentHumidity float64, pvAvailable bool) bool {
	lp.RLock()
	mode, enabled := lp.mode, lp.enabled
	switched, config := lp.dehumidifierSwitched, lp.Dehumidifier
	lp.RUnlock()

	if mode == api.ModeOff {
		return false
	}

	elapsed := lp.clock.Since(switched)
	if enabled && elapsed < config.MinOnTime {
		return true
	}
	if !enabled && elapsed < config.MinOffTime {
		return false
	}

	desired := enabled
	if currentHumidity > config.TargetHumidity+config.Hysteresis {
		desired = true
	} else if currentHumidity < config.TargetHumidity-config.Hysteresis {
		desired = false
	}

	return desired && (mode != api.ModeSmart || pvAvailable)
}

func (lp *Loadpoint) dehumidifierMinOnActive() bool {
	lp.RLock()
	enabled, switched, minOnTime := lp.enabled, lp.dehumidifierSwitched, lp.Dehumidifier.MinOnTime
	lp.RUnlock()

	return enabled && lp.clock.Since(switched) < minOnTime
}

func (lp *Loadpoint) updateDehumidifier(sitePower, batteryPower float64, batteryBuffered, batteryStart bool) error {
	mode := lp.GetMode()
	var humidity float64
	var humidityErr error
	if mode == api.ModeOff {
		lp.publish(keys.Humidity, nil)
	} else {
		humidityGetter, ok := api.Cap[api.HumidityGetter](lp.charger)
		if ok {
			humidity, humidityErr = humidityGetter.CurrentHumidity()
			if humidityErr == nil {
				if math.IsNaN(humidity) || math.IsInf(humidity, 0) || humidity < 0 || humidity > 100 {
					humidityErr = fmt.Errorf("invalid humidity reading: %v %%RH", humidity)
				} else {
					lp.publish(keys.Humidity, humidity)
				}
			}
		} else {
			humidityErr = errors.New("dehumidifier has no humidity sensor")
		}
		if humidityErr != nil {
			lp.publish(keys.Humidity, nil)
		}
	}

	desired := mode != api.ModeOff
	if desired {
		if humidityErr != nil {
			return fmt.Errorf("read dehumidifier humidity: %w", humidityErr)
		}
		desired = lp.evaluateDehumidifierState(humidity, true)
		if mode == api.ModeSmart && desired && !lp.dehumidifierMinOnActive() {
			pvCurrent := lp.pvMaxCurrent(sitePower, batteryPower, batteryBuffered, batteryStart)
			desired = lp.evaluateDehumidifierState(humidity, pvCurrent >= lp.effectiveMinCurrent())
		}
	}

	if desired == lp.enabled {
		return nil
	}
	if err := lp.charger.Enable(desired); err != nil {
		return fmt.Errorf("dehumidifier %s: %w", status[desired], err)
	}

	lp.setAndPublishEnabled(desired)
	lp.chargerSwitched = lp.clock.Now()
	return nil
}
