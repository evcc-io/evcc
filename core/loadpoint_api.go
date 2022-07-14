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

// GetMode returns loadpoint charge mode
func (lp *LoadPoint) GetMode() api.ChargeMode {
	lp.Lock()
	defer lp.Unlock()
	return lp.Mode
}

// SetMode sets loadpoint charge mode
func (lp *LoadPoint) SetMode(mode api.ChargeMode) {
	lp.Lock()
	defer lp.Unlock()

	if _, err := api.ChargeModeString(mode.String()); err != nil {
		lp.log.WARN.Printf("invalid charge mode: %s", string(mode))
		return
	}

	// apply immediately
	if lp.Mode != mode {
		lp.setMode(mode)
		lp.requestUpdate()
	}
}

// setMode sets loadpoint charge mode (lock-free)
func (lp *LoadPoint) setMode(mode api.ChargeMode) {
	lp.log.DEBUG.Printf("set charge mode: %s", string(mode))

	lp.Mode = mode
	lp.publish("mode", mode)

	// immediately allow pv mode activity
	lp.elapsePVTimer()
}

// GetTargetSoC returns loadpoint charge target soc
func (lp *LoadPoint) GetTargetSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Target
}

// SetTargetSoC sets loadpoint charge target soc
func (lp *LoadPoint) SetTargetSoC(soc int) {
	lp.Lock()
	defer lp.Unlock()

	// apply immediately
	if lp.SoC.Target != soc {
		lp.setTargetSoC(soc)
		lp.requestUpdate()
	}
}

// setTargetSoC sets loadpoint charge target soc (lock-free)
func (lp *LoadPoint) setTargetSoC(soc int) {
	lp.log.DEBUG.Println("set target soc:", soc)

	lp.SoC.Target = soc
	lp.socTimer.SoC = soc
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
	lp.Lock()
	defer lp.Unlock()

	// apply immediately
	if lp.SoC.Min != soc {
		lp.setMinSoC(soc)
		lp.requestUpdate()
	}
}

// setMinSoC sets loadpoint charge minimum soc (lock-free)
func (lp *LoadPoint) setMinSoC(soc int) {
	lp.log.DEBUG.Println("set min soc:", soc)

	lp.SoC.Min = soc
	lp.publish("minSoC", soc)
}

// GetPhases returns loadpoint enabled phases
func (lp *LoadPoint) GetPhases() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.phases
}

// SetPhases sets loadpoint enabled phases
func (lp *LoadPoint) SetPhases(phases int) error {
	// limit auto mode (phases=0) to scalable charger
	if _, ok := lp.charger.(api.ChargePhases); !ok && phases == 0 {
		return fmt.Errorf("invalid number of phases: %d", phases)
	}

	if phases != 0 && phases != 1 && phases != 3 {
		return fmt.Errorf("invalid number of phases: %d", phases)
	}

	if _, ok := lp.charger.(api.ChargePhases); ok && phases > 0 {
		return lp.scalePhases(phases)
	}

	lp.setPhases(phases)
	return nil
}

// SetTargetCharge sets loadpoint charge targetSoC
func (lp *LoadPoint) SetTargetCharge(finishAt time.Time, soc int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Printf("set target charge: %d @ %v", soc, finishAt)

	// apply immediately
	if lp.socTimer.Time != finishAt || lp.SoC.Target != soc {
		lp.socTimer.Set(finishAt)

		// don't remove soc
		if !finishAt.IsZero() {
			lp.setTargetSoC(soc)
			lp.requestUpdate()
		}
	}
}

// RemoteControl sets remote status demand
func (lp *LoadPoint) RemoteControl(source string, demand loadpoint.RemoteDemand) {
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

// GetMinCurrent returns the min loadpoint current
func (lp *LoadPoint) GetMinCurrent() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.MinCurrent
}

// SetMinCurrent sets the min loadpoint current
func (lp *LoadPoint) SetMinCurrent(current float64) {
	lp.Lock()
	defer lp.Unlock()

	if current != lp.MinCurrent {
		lp.setMinCurrent(current)
	}
}

// setMinCurrent sets the min loadpoint current (lock-free)
func (lp *LoadPoint) setMinCurrent(current float64) {
	lp.log.DEBUG.Println("set min current:", current)

	lp.MinCurrent = current
	lp.publish("minCurrent", lp.MinCurrent)
}

// GetMaxCurrent returns the max loadpoint current
func (lp *LoadPoint) GetMaxCurrent() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.MaxCurrent
}

// SetMaxCurrent returns the max loadpoint current
func (lp *LoadPoint) SetMaxCurrent(current float64) {
	lp.Lock()
	defer lp.Unlock()

	if current != lp.MaxCurrent {
		lp.setMaxCurrent(current)
	}
}

// setMaxCurrent sets the max loadpoint current (lock-free)
func (lp *LoadPoint) setMaxCurrent(current float64) {
	lp.log.DEBUG.Println("set max current:", current)

	lp.MinCurrent = current
	lp.publish("minCurrent", lp.MinCurrent)
}

// GetMinPower returns the min loadpoint power for a single phase
func (lp *LoadPoint) GetMinPower() float64 {
	return Voltage * lp.GetMinCurrent()
}

// GetMaxPower returns the max loadpoint power taking vehicle capabilities and phase scaling into account
func (lp *LoadPoint) GetMaxPower() float64 {
	return Voltage * lp.GetMaxCurrent() * float64(lp.maxActivePhases())
}

// setRemainingDuration sets the estimated remaining charging duration
func (lp *LoadPoint) setRemainingDuration(chargeRemainingDuration time.Duration) {
	if lp.chargeRemainingDuration != chargeRemainingDuration {
		lp.chargeRemainingDuration = chargeRemainingDuration
		lp.publish("chargeRemainingDuration", chargeRemainingDuration)
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
	lp.Lock()
	defer lp.Unlock()

	if lp.chargeRemainingEnergy != chargeRemainingEnergy {
		lp.chargeRemainingEnergy = chargeRemainingEnergy
		lp.publish("chargeRemainingEnergy", chargeRemainingEnergy)
	}
}

// GetRemainingEnergy is the remaining charge energy in Wh
func (lp *LoadPoint) GetRemainingEnergy() float64 {
	lp.Lock()
	defer lp.Unlock()
	return lp.chargeRemainingEnergy
}

// GetVehicles is the list of vehicles
func (lp *LoadPoint) GetVehicles() []api.Vehicle {
	lp.Lock()
	defer lp.Unlock()
	return lp.vehicles
}

// SetVehicle sets the active vehicle
func (lp *LoadPoint) SetVehicle(vehicle api.Vehicle) {
	lp.Lock()
	defer lp.Unlock()

	// set desired vehicle
	lp.setActiveVehicle(vehicle)

	// disable auto-detect
	lp.stopVehicleDetection()
}
