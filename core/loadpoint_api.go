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
	return lp.Title_
}

// GetStatus returns the charging status
func (lp *Loadpoint) GetStatus() api.ChargeStatus {
	lp.Lock()
	defer lp.Unlock()
	return lp.status
}

// GetMode returns loadpoint charge mode
func (lp *Loadpoint) GetMode() api.ChargeMode {
	lp.Lock()
	defer lp.Unlock()
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

// getChargedEnergy returns loadpoint charge target energy in Wh
func (lp *Loadpoint) getChargedEnergy() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.sessionEnergy.TotalWh()
}

// GetTargetEnergy returns loadpoint charge target energy
func (lp *Loadpoint) GetTargetEnergy() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.targetEnergy
}

// setTargetEnergy sets loadpoint charge target energy (no mutex)
func (lp *Loadpoint) setTargetEnergy(energy float64) {
	lp.targetEnergy = energy
	lp.publish(targetEnergy, energy)
}

// SetTargetEnergy sets loadpoint charge target energy
func (lp *Loadpoint) SetTargetEnergy(energy float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set target energy:", energy)

	// apply immediately
	if lp.targetEnergy != energy {
		lp.setTargetEnergy(energy)
		lp.requestUpdate()
		lp.persistVehicleSettings()
	}
}

// GetPriority returns the loadpoint priority
func (lp *Loadpoint) GetPriority() int {
	lp.Lock()
	defer lp.Unlock()
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

// GetTargetSoc returns loadpoint charge target soc
func (lp *Loadpoint) GetTargetSoc() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.Soc.target
}

// setTargetSoc sets loadpoint charge target soc (no mutex)
func (lp *Loadpoint) setTargetSoc(soc int) {
	lp.Soc.target = soc
	lp.publish(targetSoc, soc)
}

// SetTargetSoc sets loadpoint charge target soc
func (lp *Loadpoint) SetTargetSoc(soc int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set target soc:", soc)

	// apply immediately
	if lp.Soc.target != soc {
		lp.setTargetSoc(soc)
		lp.requestUpdate()
		lp.persistVehicleSettings()
	}
}

// GetMinSoc returns loadpoint charge minimum soc
func (lp *Loadpoint) GetMinSoc() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.Soc.min
}

// setMinSoc sets loadpoint charge min soc (no mutex)
func (lp *Loadpoint) setMinSoc(soc int) {
	lp.Soc.min = soc
	lp.publish(minSoc, soc)
}

// SetMinSoc sets loadpoint charge minimum soc
func (lp *Loadpoint) SetMinSoc(soc int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set min soc:", soc)

	// apply immediately
	if lp.Soc.min != soc {
		lp.setMinSoc(soc)
		lp.requestUpdate()
		lp.persistVehicleSettings()
	}
}

// GetPhases returns loadpoint enabled phases
func (lp *Loadpoint) GetPhases() int {
	lp.Lock()
	defer lp.Unlock()
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

// GetTargetTime returns the target time
func (lp *Loadpoint) GetTargetTime() time.Time {
	lp.Lock()
	defer lp.Unlock()
	return lp.targetTime
}

// SetTargetTime sets the charge target time
func (lp *Loadpoint) SetTargetTime(finishAt time.Time) error {
	if !finishAt.IsZero() && finishAt.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set target time:", finishAt.Round(time.Second).Local())

	lp.setTargetTime(finishAt)
	lp.persistVehicleSettings()

	return nil
}

// setTargetTime sets the charge target time
func (lp *Loadpoint) setTargetTime(finishAt time.Time) {
	lp.targetTime = finishAt
	lp.publish(targetTime, finishAt)

	// TODO planActive is not guarded by mutex
	if finishAt.IsZero() {
		lp.setPlanActive(false)
	}
}

// GetEnableThreshold gets the loadpoint enable threshold
func (lp *Loadpoint) GetEnableThreshold() float64 {
	lp.Lock()
	defer lp.Unlock()
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
	lp.Lock()
	defer lp.Unlock()
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
	lp.Lock()
	defer lp.Unlock()
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
	lp.Lock()
	defer lp.Unlock()
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
	lp.Lock()
	defer lp.Unlock()
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

// GetPlanActive returns the active state of the planner
func (lp *Loadpoint) GetPlanActive() bool {
	lp.Lock()
	defer lp.Unlock()
	return lp.planActive
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

// GetRemainingDuration is the estimated remaining charging duration
func (lp *Loadpoint) GetRemainingDuration() time.Duration {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargeRemainingDuration
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

// GetRemainingEnergy is the remaining charge energy in Wh
func (lp *Loadpoint) GetRemainingEnergy() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargeRemainingEnergy
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
