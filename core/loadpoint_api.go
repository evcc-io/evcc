package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/wrapper"
)

var _ loadpoint.API = (*LoadPoint)(nil)

// GetStatus returns the charging status
func (lp *LoadPoint) GetStatus() api.ChargeStatus {
	lp.Lock()
	defer lp.Unlock()
	return lp.status
}

// charging returns the EVs charging state
func (lp *LoadPoint) setStatus(status api.ChargeStatus) {
	lp.Lock()
	defer lp.Unlock()
	lp.status = status
}

// GetMode returns loadpoint charge mode
func (lp *LoadPoint) GetMode() api.ChargeMode {
	lp.Lock()
	defer lp.Unlock()
	return lp.Mode
}

// SetMode sets loadpoint charge mode
func (lp *LoadPoint) SetMode(mode api.ChargeMode) {
	if _, err := api.ChargeModeString(mode.String()); err != nil {
		lp.log.WARN.Printf("invalid charge mode: %s", string(mode))
		return
	}

	lp.log.DEBUG.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if lp.GetMode() != mode {
		lp.setMode(mode)

		// immediately allow pv mode activity
		// TODO sync pv timers
		lp.elapsePVTimer()

		lp.requestUpdate()
	}
}

func (lp *LoadPoint) setMode(mode api.ChargeMode) {
	lp.Lock()
	lp.Mode = mode
	lp.Unlock()
	lp.publish("mode", mode)
}

// GetTargetSoC returns loadpoint charge target soc
func (lp *LoadPoint) GetTargetSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Target
}

// SetTargetSoC sets loadpoint charge target soc
func (lp *LoadPoint) SetTargetSoC(soc int) {
	lp.log.DEBUG.Println("set target soc:", soc)

	if lp.GetTargetSoC() != soc {
		lp.setTargetSoC(soc)
		lp.requestUpdate()
	}
}

func (lp *LoadPoint) setTargetSoC(soc int) {
	lp.Lock()
	lp.SoC.Target = soc
	lp.Unlock()

	lp.publish("targetSoC", soc)
}

// GetMinSoC returns loadpoint charge minimum soc
func (lp *LoadPoint) GetMinSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Min
}

// SetMinSoC sets loadpoint charge minimum soc
func (lp *LoadPoint) SetMinSoC(soc int) {
	lp.log.DEBUG.Println("set min soc:", soc)

	// apply immediately
	if lp.GetMinSoC() != soc {
		lp.Lock()
		lp.SoC.Min = soc
		lp.Unlock()

		lp.publish("minSoC", soc)
		lp.requestUpdate()
	}
}

// GetPhases returns loadpoint enabled phases
func (lp *LoadPoint) GetPhases() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.Phases
}

// SetPhases sets loadpoint enabled phases
func (lp *LoadPoint) SetPhases(phases int) error {
	if phases != 1 && phases != 3 {
		return fmt.Errorf("invalid number of phases: %d", phases)
	}

	lp.log.DEBUG.Println("set phases:", phases)

	// TODO sync scalephases
	if _, ok := lp.charger.(api.ChargePhases); ok {
		return lp.scalePhases(phases)
	}

	lp.setPhases(phases)
	return nil
}

// setPhases sets the number of enabled phases without modifying the charger
func (lp *LoadPoint) setPhases(phases int) {
	if lp.GetPhases() != phases {
		lp.Lock()
		lp.Phases = phases
		lp.phaseTimer = time.Time{}
		lp.Unlock()

		lp.publish("phases", phases)

		// TODO sync phase timer
		lp.publishTimer(phaseTimer, 0, timerInactive)

		lp.setMeasuredPhases(0)
	}
}

// SetTargetCharge sets loadpoint charge targetSoC
func (lp *LoadPoint) SetTargetCharge(targetTime time.Time, soc int) {
	lp.log.DEBUG.Printf("set target charge: %d @ %v", soc, targetTime)

	// apply immediately
	if lp.GetTargetTime() != targetTime || lp.GetTargetSoC() != soc {
		lp.SetTargetTime(targetTime)

		// don't remove soc
		if !targetTime.IsZero() {
			lp.setTargetSoC(soc)
			lp.publish("targetTimeHourSuggestion", targetTime.Hour())
			lp.requestUpdate()
		}
	}
}

func (lp *LoadPoint) GetTargetTime() time.Time {
	lp.Lock()
	defer lp.Unlock()
	return lp.targetTime
}

func (lp *LoadPoint) SetTargetTime(targetTime time.Time) {
	lp.Lock()
	lp.targetTime = targetTime
	lp.Unlock()

	lp.publish("targetTime", targetTime)
}

// SetVehicle sets the active vehicle
func (lp *LoadPoint) SetVehicle(vehicle api.Vehicle) {
	title := unknownVehicle
	if vehicle != nil {
		title = vehicle.Title()
	}
	lp.log.DEBUG.Println("set vehicle:", title)

	if lp.getVehicle() != vehicle {
		lp.setVehicle(vehicle)
	}
}

// RemoteControl sets remote status demand
func (lp *LoadPoint) RemoteControl(source string, demand loadpoint.RemoteDemand) {
	lp.log.DEBUG.Println("remote demand:", demand)

	// apply immediately
	if lp.getRemoteDemand() != demand {
		lp.setRemoteDemand(demand, source)
		lp.requestUpdate()
	}
}

func (lp *LoadPoint) getRemoteDemand() loadpoint.RemoteDemand {
	lp.Lock()
	defer lp.Unlock()
	return lp.remoteDemand
}

func (lp *LoadPoint) setRemoteDemand(demand loadpoint.RemoteDemand, source string) {
	lp.Lock()
	lp.remoteDemand = demand
	lp.Unlock()

	lp.publish("remoteDisabled", demand)
	lp.publish("remoteDisabledSource", source)
}

// HasChargeMeter determines if a physical charge meter is attached
func (lp *LoadPoint) HasChargeMeter() bool {
	_, isWrapped := lp.chargeMeter.(*wrapper.ChargeMeter)
	return lp.chargeMeter != nil && !isWrapped
}

// GetChargePower returns the current charge power
func (lp *LoadPoint) GetChargePower() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargePower
}

func (lp *LoadPoint) setChargePower(power float64) {
	lp.Lock()
	lp.chargePower = power
	lp.Unlock()

	lp.publish("chargePower", power)
}

// GetMinCurrent returns the min loadpoint current
func (lp *LoadPoint) GetMinCurrent() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.MinCurrent
}

// SetMinCurrent returns the min loadpoint current
func (lp *LoadPoint) SetMinCurrent(current float64) {
	lp.log.DEBUG.Println("set min current:", current)

	if lp.GetMinCurrent() != current {
		lp.setMinCurrent(current)
		lp.requestUpdate()
	}
}

func (lp *LoadPoint) setMinCurrent(current float64) {
	lp.Lock()
	lp.MinCurrent = current
	lp.Unlock()

	lp.publish("minCurrent", current)
}

// GetMaxCurrent returns the max loadpoint current
func (lp *LoadPoint) GetMaxCurrent() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.MaxCurrent
}

// SetMaxCurrent returns the max loadpoint current
func (lp *LoadPoint) SetMaxCurrent(current float64) {
	lp.log.DEBUG.Println("set max current:", current)

	if lp.GetMaxCurrent() != current {
		lp.setMaxCurrent(current)
		lp.requestUpdate()
	}
}

func (lp *LoadPoint) setMaxCurrent(current float64) {
	lp.Lock()
	lp.MaxCurrent = current
	lp.Unlock()

	lp.publish("maxCurrent", current)
}

// GetMinPower returns the min loadpoint power for a single phase
func (lp *LoadPoint) GetMinPower() float64 {
	return Voltage * lp.GetMinCurrent()
}

// GetMaxPower returns the max loadpoint power taking vehicle capabilities and phase scaling into account
func (lp *LoadPoint) GetMaxPower() float64 {
	return Voltage * lp.GetMaxCurrent() * float64(lp.activePhases(true))
}

// setRemainingDuration sets the estimated remaining charging duration
func (lp *LoadPoint) setRemainingDuration(chargeRemainingDuration time.Duration) {
	if lp.GetRemainingDuration() != chargeRemainingDuration {
		lp.Lock()
		lp.chargeRemainingDuration = chargeRemainingDuration
		lp.Unlock()

		if chargeRemainingDuration <= 0 {
			lp.publish("chargeRemainingDuration", nil)
		} else {
			lp.publish("chargeRemainingDuration", chargeRemainingDuration)
		}
	}
}

// GetRemainingDuration is the estimated remaining charging duration
func (lp *LoadPoint) GetRemainingDuration() time.Duration {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargeRemainingDuration
}

// setRemainingEnergy sets the remaining charge energy in Wh
func (lp *LoadPoint) setRemainingEnergy(chargeRemainingEnergy float64) {
	if lp.GetRemainingEnergy() != chargeRemainingEnergy {
		lp.Lock()
		lp.chargeRemainingEnergy = chargeRemainingEnergy
		lp.Unlock()

		if chargeRemainingEnergy <= 0 {
			lp.publish("chargeRemainingEnergy", nil)
		} else {
			lp.publish("chargeRemainingEnergy", chargeRemainingEnergy)
		}
	}
}

// GetRemainingEnergy is the remaining charge energy in Wh
func (lp *LoadPoint) GetRemainingEnergy() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargeRemainingEnergy
}
