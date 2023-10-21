package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
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
		lp.publish("mode", mode)

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

// GetPlanEnergy returns plan target energy
func (lp *Loadpoint) GetPlanEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.planEnergy
}

// setPlanEnergy sets plan target energy (no mutex)
func (lp *Loadpoint) setPlanEnergy(energy float64) {
	lp.planEnergy = energy
	lp.publish(planEnergy, energy)
}

// SetPlanEnergy sets plan target energy
func (lp *Loadpoint) SetPlanEnergy(energy float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set target energy:", energy)

	// apply immediately
	if lp.planEnergy != energy {
		lp.setPlanEnergy(energy)
		lp.requestUpdate()
		lp.persistVehicleSettings()
	}
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
		lp.publish("priority", prio)
	}
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
	// TODO decide name
	lp.publish(limitSoc, soc)
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
		lp.persistVehicleSettings()
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

// GetPlanSoc returns the plan soc
func (lp *Loadpoint) GetPlanSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.planSoc
}

// SetPlanSoc sets the plan soc
func (lp *Loadpoint) SetPlanSoc(soc int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set plan soc:", soc)

	if lp.planSoc != soc {
		lp.planSoc = soc
		lp.publish("planSoc", soc)
	}
}

// GetPlanTime returns the plan time
func (lp *Loadpoint) GetPlanTime() time.Time {
	lp.RLock()
	defer lp.RUnlock()
	return lp.planTime
}

// SetPlanTime sets the charge plan time
func (lp *Loadpoint) SetPlanTime(finishAt time.Time) error {
	if !finishAt.IsZero() && finishAt.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	lp.Lock()
	defer lp.Unlock()
	lp.setPlanTime(finishAt)
	lp.persistVehicleSettings()

	return nil
}

// setPlanTime sets the charge plan time
func (lp *Loadpoint) setPlanTime(finishAt time.Time) {
	lp.planTime = finishAt
	lp.publish(planTime, finishAt)

	// TODO planActive is not guarded by mutex
	if finishAt.IsZero() {
		lp.setPlanActive(false)
	}
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

		lp.publish("remoteDisabled", demand)
		lp.publish("remoteDisabledSource", source)

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
		lp.publish(minCurrent, lp.MinCurrent)
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
		lp.publish(maxCurrent, lp.MaxCurrent)
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
		lp.publish(chargeRemainingDuration, remainingDuration)
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
		lp.publish("chargeRemainingEnergy", chargeRemainingEnergy)
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
