package core

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
)

//go:generate mockgen -package mock -destination ../mock/mock_chargerhandler.go github.com/andig/evcc/core Handler

// Handler is the charger handler responsible for enabled state, target current and guard durations
type Handler interface {
	Prepare()
	SyncEnabled()
	Enabled() bool
	Status() (api.ChargeStatus, error)
	TargetCurrent() int64
	Ramp(int64, ...bool) error
}

// HandlerConfig contains the public configuration for the ChargerHandler
type HandlerConfig struct {
	Sensitivity   int64         // Step size of current change
	MinCurrent    int64         // PV mode: start current	Min+PV mode: min current
	MaxCurrent    int64         // Max allowed current. Physically ensured by the charge controller
	GuardDuration time.Duration // charger enable/disable minimum holding time
}

// ChargerHandler handles steering of the charger state and allowed current
type ChargerHandler struct {
	clock clock.Clock // mockable time
	bus   evbus.Bus   // event bus
	log   *util.Logger

	charger api.Charger // Charger

	HandlerConfig // public configuration

	enabled       bool  // Charger enabled state
	targetCurrent int64 // Charger target current

	// contactor switch guard
	guardUpdated time.Time // charger enabled/disabled timestamp
}

// Status returns charger status
func (lp *ChargerHandler) Status() (api.ChargeStatus, error) {
	return lp.charger.Status()
}

// Enabled returns handler enabled state
func (lp *ChargerHandler) Enabled() bool {
	return lp.enabled
}

// TargetCurrent returns handler target current
func (lp *ChargerHandler) TargetCurrent() int64 {
	return lp.targetCurrent
}

// Prepare synchronizes initial charger enabled state and current
func (lp *ChargerHandler) Prepare() {
	// read initial enabled state
	enabled, err := lp.charger.Enabled()
	if err == nil {
		lp.enabled = enabled
		lp.log.INFO.Printf("charger %sd", status[lp.enabled])

		// prevent immediately disabling charger
		if lp.enabled {
			lp.guardUpdated = lp.clock.Now()
		}
	} else {
		lp.log.ERROR.Printf("charger error: %v", err)
	}

	// set current to known value
	if err = lp.setTargetCurrent(lp.MinCurrent); err != nil {
		lp.log.ERROR.Println(err)
	}
	lp.bus.Publish(evChargeCurrent, lp.MinCurrent)
}

// SyncEnabled synchronizes charger settings to expected state
func (lp *ChargerHandler) SyncEnabled() {
	enabled, err := lp.charger.Enabled()
	if err == nil && enabled != lp.enabled {
		lp.log.WARN.Printf("sync enabled state to %s", status[lp.enabled])
		err = lp.charger.Enable(lp.enabled)
	}

	if err != nil {
		lp.log.ERROR.Printf("charge controller error: %v", err)
	}
}

// chargerEnable switches charging on or off. Minimum cycle duration is guaranteed.
func (lp *ChargerHandler) chargerEnable(enable bool) error {
	if lp.targetCurrent != 0 && lp.targetCurrent != lp.MinCurrent {
		lp.log.FATAL.Fatal("charger enable/disable called without setting min current first")
	}

	if remaining := (lp.GuardDuration - lp.clock.Since(lp.guardUpdated)).Truncate(time.Second); remaining > 0 {
		lp.log.DEBUG.Printf("charger %s - contactor delay %v", status[enable], remaining)
		return nil
	}

	if lp.enabled != enable {
		if err := lp.charger.Enable(enable); err != nil {
			return fmt.Errorf("charge controller error: %v", err)
		}

		lp.enabled = enable // cache
		lp.log.INFO.Printf("charger %s", status[enable])
		lp.guardUpdated = lp.clock.Now()
	} else {
		lp.log.DEBUG.Printf("charger %s", status[enable])
	}

	// if not enabled, current will be reduced to 0 in handler
	lp.bus.Publish(evChargeCurrent, lp.MinCurrent)

	return nil
}

// setTargetCurrent guards setting current against changing to identical value
// and violating MaxCurrent
func (lp *ChargerHandler) setTargetCurrent(targetCurrentIn int64) error {
	targetCurrent := clamp(targetCurrentIn, lp.MinCurrent, lp.MaxCurrent)
	if targetCurrent != targetCurrentIn {
		lp.log.WARN.Printf("hard limit charge current: %dA", targetCurrent)
	}

	if lp.targetCurrent != targetCurrent {
		lp.log.DEBUG.Printf("set charge current: %dA", targetCurrent)
		if err := lp.charger.MaxCurrent(targetCurrent); err != nil {
			return fmt.Errorf("charge controller error: %v", err)
		}

		lp.targetCurrent = targetCurrent // cache
	}

	// if not enabled, current will be reduced to 0 in handler
	lp.bus.Publish(evChargeCurrent, targetCurrent)

	return nil
}

// rampUpDown moves stepwise towards target current.
// It does not enable or disable the charger.
func (lp *ChargerHandler) rampUpDown(target int64) error {
	current := lp.targetCurrent
	if current == target {
		return nil
	}

	var step int64
	if current < target {
		step = min(current+lp.Sensitivity, target)
	} else if current > target {
		step = max(current-lp.Sensitivity, target)
	}

	step = clamp(step, lp.MinCurrent, lp.MaxCurrent)

	return lp.setTargetCurrent(step)
}

// rampOff disables charger after setting minCurrent.
// Setting current and disabling are two steps. If already disabled, this is a nop.
func (lp *ChargerHandler) rampOff() error {
	if lp.enabled {
		if lp.targetCurrent != lp.MinCurrent {
			return lp.setTargetCurrent(lp.MinCurrent)
		}

		return lp.chargerEnable(false)
	}

	return nil
}

// rampOn enables charger immediately after setting minCurrent.
// If already enabled, target will be set.
func (lp *ChargerHandler) rampOn(target int64) error {
	if !lp.enabled {
		if err := lp.setTargetCurrent(lp.MinCurrent); err != nil {
			return err
		}

		return lp.chargerEnable(true)
	}

	return lp.setTargetCurrent(target)
}

// Ramp performs ramping charger current up and down where targetCurrent=0
// signals disabled state
func (lp *ChargerHandler) Ramp(targetCurrent int64, force ...bool) error {
	// reset guard updated
	if len(force) == 1 && force[0] {
		lp.guardUpdated = time.Time{}
	}

	// if targetCurrent == 0 ramp down to disabled state
	if targetCurrent == 0 {
		return lp.rampOff()
	}

	// targetCurrent != 0 and not enabled ramp to enabled state
	if !lp.enabled {
		return lp.rampOn(targetCurrent)
	}

	return lp.rampUpDown(targetCurrent)
}
