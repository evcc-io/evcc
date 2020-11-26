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
	Sync()
	Enabled() bool
	Status() (api.ChargeStatus, error)
	TargetCurrent() int64
	Ramp(int64, bool) error
}

// HandlerConfig contains the public configuration for the ChargerHandler
type HandlerConfig struct {
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

// Sync synchronizes charger settings to expected state
func (lp *ChargerHandler) Sync() {
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
func (lp *ChargerHandler) setTargetCurrent(targetCurrent int64) error {
	target := clamp(targetCurrent, lp.MinCurrent, lp.MaxCurrent)

	if lp.targetCurrent != target {
		lp.log.DEBUG.Printf("set charge current: %dA", target)
		if err := lp.charger.MaxCurrent(target); err != nil {
			return fmt.Errorf("charge controller error: %v", err)
		}

		lp.targetCurrent = target // cache
	}

	// if not enabled, current will be reduced to 0 in handler
	lp.bus.Publish(evChargeCurrent, target)

	return nil
}

// Ramp performs ramping charger current up and down where targetCurrent=0
// signals disabled state
func (lp *ChargerHandler) Ramp(targetCurrent int64, force bool) error {
	// reset guard updated
	if force {
		lp.guardUpdated = time.Time{}
	}

	// if targetCurrent == 0 disable
	if targetCurrent == 0 {
		err := lp.chargerEnable(false)
		if err == nil && lp.enabled && lp.targetCurrent > lp.MinCurrent {
			err = lp.setTargetCurrent(lp.MinCurrent)
		}

		return err
	}

	// else set targetCurrent and optionally enable
	err := lp.setTargetCurrent(targetCurrent)
	if err == nil && !lp.enabled {
		err = lp.chargerEnable(true)
	}

	return err
}
