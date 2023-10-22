package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/wrapper"
)

var _ loadpoint.API = (*Loadpoint)(nil)

// Title returns the human-readable loadpoint title
func (lp *Loadpoint) Title() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Title_
}

// GetStatus returns the charging status
func (lp *Loadpoint) GetStatus() api.ChargeStatus {
	lp.RLock()
	defer lp.RUnlock()
	return lp.status
}

// GetMode returns loadpoint charge mode
func (lp *Loadpoint) GetMode() api.ChargeMode {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Mode
}

// SetMode sets loadpoint charge mode
func (lp *Loadpoint) SetMode(mode api.ChargeMode) {
	lp.Lock()
	defer lp.Unlock()

	if _, err := api.ChargeModeString(mode.String()); err != nil {
		lp.log.ERROR.Printf("invalid charge mode: %s", string(mode))
		return
	}

	lp.log.DEBUG.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if lp.Mode != mode {
		lp.Mode = mode
		lp.publish(keys.Mode, mode)

		// reset timers
		switch mode {
		case api.ModeNow, api.ModeOff:
			lp.resetPhaseTimer()
			lp.resetPVTimer()
			lp.setPlanActive(false)
		case api.ModeMinPV:
			lp.resetPVTimer()
		}

		lp.requestUpdate()
	}
}

// getChargedEnergy returns plan target energy in Wh
func (lp *Loadpoint) getChargedEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.sessionEnergy.TotalWh()
}

// GetPriority returns the loadpoint priority
func (lp *Loadpoint) GetPriority() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Priority_
}

// SetPriority sets the loadpoint priority
func (lp *Loadpoint) SetPriority(prio int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set priority:", prio)

	if lp.Priority_ != prio {
		lp.Priority_ = prio
		lp.publish(keys.Priority, prio)
	}
}

// GetPhases returns loadpoint enabled phases
func (lp *Loadpoint) GetPhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.phases
}

// SetPhases sets loadpoint enabled phases
func (lp *Loadpoint) SetPhases(phases int) error {
	// limit auto mode (phases=0) to scalable charger
	if _, ok := lp.charger.(api.PhaseSwitcher); !ok && phases == 0 {
		return fmt.Errorf("invalid number of phases: %d", phases)
	}

	if phases != 0 && phases != 1 && phases != 3 {
		return fmt.Errorf("invalid number of phases: %d", phases)
	}

	// set new default
	lp.log.DEBUG.Println("set phases:", phases)
	lp.setConfiguredPhases(phases)

	// apply immediately if not 1p3p
	if _, ok := lp.charger.(api.PhaseSwitcher); !ok {
		lp.setPhases(phases)
	}

	lp.requestUpdate()

	return nil
}

// GetSessionLimitSoc returns the session limit soc
func (lp *Loadpoint) GetSessionLimitSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.sessionLimitSoc
}

// setSessionLimitSoc sets the session limit soc (no mutex)
func (lp *Loadpoint) setSessionLimitSoc(soc int) {
	lp.sessionLimitSoc = soc
	lp.publish(keys.LimitSoc, soc)
}

// SetSessionLimitSoc sets the session soc limit
func (lp *Loadpoint) SetSessionLimitSoc(soc int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set session soc limit:", soc)

	// apply immediately
	if lp.sessionLimitSoc != soc {
		lp.setSessionLimitSoc(soc)
		lp.requestUpdate()
		lp.settings.SetInt(keys.LimitSoc, int64(soc))
	}
}

// GetSessionLimitEnergy returns the session limit energy
func (lp *Loadpoint) GetSessionLimitEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.sessionLimitEnergy
}

// setSessionLimitEnergy sets the session limit energy (no mutex)
func (lp *Loadpoint) setSessionLimitEnergy(energy float64) {
	lp.sessionLimitEnergy = energy
	lp.publish(keys.LimitEnergy, energy)
}

// SetSessionLimitEnergy sets the session energy limit
func (lp *Loadpoint) SetSessionLimitEnergy(energy float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set session energy limit:", energy)

	// apply immediately
	if lp.sessionLimitEnergy != energy {
		lp.setSessionLimitEnergy(energy)
		lp.requestUpdate()
		lp.settings.SetFloat(keys.LimitEnergy, energy)
	}
}

// GetPlanTime returns the plan time
func (lp *Loadpoint) GetPlanTime() time.Time {
	lp.RLock()
	defer lp.RUnlock()
	return lp.planTime
}

// setPlanTime sets the charge plan time
func (lp *Loadpoint) setPlanTime(finishAt time.Time) {
	lp.planTime = finishAt
	lp.publish(keys.PlanTime, finishAt)

	if finishAt.IsZero() {
		lp.setPlanActive(false)
	}
}

// GetPlanEnergy returns plan target energy
func (lp *Loadpoint) GetPlanEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.planEnergy
}

// setPlanEnergy sets plan target energy (no mutex)
func (lp *Loadpoint) setPlanEnergy(energy float64) {
	lp.planEnergy = energy
	lp.publish(keys.PlanEnergy, energy)

	if energy == 0 {
		lp.setPlanActive(false)
	}
}

// SetPlanEnergy sets plan target energy
func (lp *Loadpoint) SetPlanEnergy(finishAt time.Time, energy float64) error {
	lp.Lock()
	defer lp.Unlock()

	if !finishAt.IsZero() && finishAt.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	lp.log.DEBUG.Println("set plan energy:", energy)

	// apply immediately
	if lp.planEnergy != energy || !lp.planTime.Equal(finishAt) {
		lp.setPlanEnergy(energy)
		lp.setPlanTime(finishAt)
		lp.requestUpdate()

		// TODO decide persisting in api vs internal setter
		lp.settings.SetFloat(keys.PlanEnergy, energy)
		lp.settings.SetTime(keys.PlanTime, finishAt)
	}

	return nil
}

// GetEnableThreshold gets the loadpoint enable threshold
func (lp *Loadpoint) GetEnableThreshold() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Enable.Threshold
}

// SetEnableThreshold sets loadpoint enable threshold
func (lp *Loadpoint) SetEnableThreshold(threshold float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set enable threshold:", threshold)

	if lp.Enable.Threshold != threshold {
		lp.Enable.Threshold = threshold
		lp.publish("enableThreshold", threshold)
	}
}

// GetDisableThreshold gets the loadpoint enable threshold
func (lp *Loadpoint) GetDisableThreshold() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Disable.Threshold
}

// SetDisableThreshold sets loadpoint disable threshold
func (lp *Loadpoint) SetDisableThreshold(threshold float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set disable threshold:", threshold)

	if lp.Disable.Threshold != threshold {
		lp.Disable.Threshold = threshold
		lp.publish("disableThreshold", threshold)
	}
}

// RemoteControl sets remote status demand
func (lp *Loadpoint) RemoteControl(source string, demand loadpoint.RemoteDemand) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("remote demand:", demand)

	// apply immediately
	if lp.remoteDemand != demand {
		lp.remoteDemand = demand

		lp.publish(keys.RemoteDisabled, demand)
		lp.publish(keys.RemoteDisabledSource, source)

		lp.requestUpdate()
	}
}

// HasChargeMeter determines if a physical charge meter is attached
func (lp *Loadpoint) HasChargeMeter() bool {
	_, isWrapped := lp.chargeMeter.(*wrapper.ChargeMeter)
	return lp.chargeMeter != nil && !isWrapped
}

// GetChargePower returns the current charge power
func (lp *Loadpoint) GetChargePower() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.chargePower
}

// GetChargePowerFlexibility returns the flexible amount of current charging power
func (lp *Loadpoint) GetChargePowerFlexibility() float64 {
	// no locking
	mode := lp.GetMode()
	if mode == api.ModeNow || !lp.charging() || lp.minSocNotReached() {
		return 0
	}

	if mode == api.ModePV {
		return lp.GetChargePower()
	}

	// MinPV mode
	return max(0, lp.GetChargePower()-lp.GetMinPower())
}

// GetMinCurrent returns the min loadpoint current
func (lp *Loadpoint) GetMinCurrent() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.MinCurrent
}

// SetMinCurrent sets the min loadpoint current
func (lp *Loadpoint) SetMinCurrent(current float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set min current:", current)

	if current != lp.MinCurrent {
		lp.MinCurrent = current
		lp.publish(keys.MinCurrent, lp.MinCurrent)
	}
}

// GetMaxCurrent returns the max loadpoint current
func (lp *Loadpoint) GetMaxCurrent() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.MaxCurrent
}

// SetMaxCurrent sets the max loadpoint current
func (lp *Loadpoint) SetMaxCurrent(current float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set max current:", current)

	if current != lp.MaxCurrent {
		lp.MaxCurrent = current
		lp.publish(keys.MaxCurrent, lp.MaxCurrent)
	}
}

// GetMinPower returns the min loadpoint power for a single phase
func (lp *Loadpoint) GetMinPower() float64 {
	return Voltage * lp.GetMinCurrent()
}

// GetMaxPower returns the max loadpoint power taking vehicle capabilities and phase scaling into account
func (lp *Loadpoint) GetMaxPower() float64 {
	return Voltage * lp.GetMaxCurrent() * float64(lp.maxActivePhases())
}

// GetRemainingDuration is the estimated remaining charging duration
func (lp *Loadpoint) GetRemainingDuration() time.Duration {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargeRemainingDuration
}

// SetRemainingDuration sets the estimated remaining charging duration
func (lp *Loadpoint) SetRemainingDuration(chargeRemainingDuration time.Duration) {
	lp.Lock()
	defer lp.Unlock()
	lp.setRemainingDuration(chargeRemainingDuration)
}

// setRemainingDuration sets the estimated remaining charging duration (no mutex)
func (lp *Loadpoint) setRemainingDuration(remainingDuration time.Duration) {
	if lp.chargeRemainingDuration != remainingDuration {
		lp.chargeRemainingDuration = remainingDuration
		lp.publish(keys.ChargeRemainingDuration, remainingDuration)
	}
}

// GetRemainingEnergy is the remaining charge energy in Wh
func (lp *Loadpoint) GetRemainingEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.chargeRemainingEnergy
}

// SetRemainingEnergy sets the remaining charge energy in Wh
func (lp *Loadpoint) SetRemainingEnergy(chargeRemainingEnergy float64) {
	lp.Lock()
	defer lp.Unlock()
	lp.setRemainingEnergy(chargeRemainingEnergy)
}

// setRemainingEnergy sets the remaining charge energy in Wh (no mutex)
func (lp *Loadpoint) setRemainingEnergy(chargeRemainingEnergy float64) {
	if lp.chargeRemainingEnergy != chargeRemainingEnergy {
		lp.chargeRemainingEnergy = chargeRemainingEnergy
		lp.publish(keys.ChargeRemainingEnergy, chargeRemainingEnergy)
	}
}

// GetVehicle gets the active vehicle
func (lp *Loadpoint) GetVehicle() api.Vehicle {
	lp.vehicleMux.Lock()
	defer lp.vehicleMux.Unlock()
	return lp.vehicle
}

// SetVehicle sets the active vehicle
func (lp *Loadpoint) SetVehicle(vehicle api.Vehicle) {
	// set desired vehicle (protected by lock, no locking here)
	lp.setActiveVehicle(vehicle)

	lp.vehicleMux.Lock()
	defer lp.vehicleMux.Unlock()

	// disable auto-detect
	lp.stopVehicleDetection()
}

// StartVehicleDetection allows triggering vehicle detection for debugging purposes
func (lp *Loadpoint) StartVehicleDetection() {
	// reset vehicle
	lp.setActiveVehicle(nil)

	lp.Lock()
	defer lp.Unlock()

	// start auto-detect
	lp.startVehicleDetection()
}
