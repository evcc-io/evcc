package core

import (
	"math"

	"github.com/evcc-io/evcc/api"
)

// CoordinatedPVBattery enables PV-only vehicle starts with the existing site buffer reserve.
func (site *Site) CoordinatedPVBattery() bool {
	for _, dev := range site.batteryMeters {
		m := dev.Instance()
		if api.HasCap[api.BatteryControlStateReader](m) && api.HasCap[api.BatteryControlRestorer](m) && api.HasCap[api.BatteryChargePowerController](m) {
			return true
		}
	}
	return false
}

// pendingVehiclePower reserves accepted current offers until meters include their load.
func (site *Site) pendingVehiclePower() float64 {
	return site.pendingVehiclePowerExcept(nil)
}

func (site *Site) pendingVehiclePowerExcept(except *Loadpoint) float64 {
	var pending float64
	for _, lp := range site.loadpoints {
		if lp == except || !lp.enabled || !lp.connected() {
			continue
		}
		power := currentToPower(lp.offeredCurrent, lp.ActivePhases())
		if !finiteBatteryPVValue(power) || !finiteBatteryPVValue(lp.chargePower) {
			return math.Inf(1)
		}
		pending += max(0, power-max(0, lp.chargePower))
	}
	return pending
}

func (site *Site) pendingBatteryPower() float64 {
	site.batteryPVMu.Lock()
	defer site.batteryPVMu.Unlock()
	var pending float64
	for _, dev := range site.batteryMeters {
		j := site.batteryPVJournals[dev.Config().Name]
		if j == nil || j.ReservedPower() <= 0 {
			continue
		}
		power, err := dev.Instance().CurrentPower()
		if err != nil || !finiteBatteryPVValue(power) {
			return math.Inf(1)
		}
		pending += max(0, j.ReservedPower()-max(0, -power))
	}
	return pending
}

// pvVehiclePriority releases only journal-owned charging before an EV can claim it.
// Foreign manual charging remains fixed consumption in the vehicle's site budget.
func (site *Site) pvVehiclePriority(state siteState) bool {
	if !site.CoordinatedPVBattery() || !finiteBatteryPVValue(state.gridPower) || state.battery.Soc < site.GetPrioritySoc() {
		return false
	}
	var ownCharge float64
	for _, dev := range site.batteryMeters {
		j := site.batteryPVJournals[dev.Config().Name]
		if j == nil || (j.Record.Stage != "active" && j.Record.Stage != "intent" && j.Record.Stage != "releasing") {
			continue
		}
		power, err := dev.Instance().CurrentPower()
		if err != nil || !finiteBatteryPVValue(power) {
			return false
		}
		ownCharge += max(0, -power)
	}
	available := -state.gridPower + ownCharge - max(100, site.GetResidualPower())
	for _, lp := range site.loadpoints {
		if !lp.coordinatedPV() || lp.GetMode() != api.ModeSmart || !lp.connected() || lp.LimitSocReached() || lp.LimitEnergyReached() {
			continue
		}
		if lp.charging() {
			if available > 0 && lp.offeredCurrent < lp.effectiveMaxCurrent() {
				return true
			}
		} else if available >= currentToPower(lp.effectiveMinCurrent(), lp.ActivePhases()) {
			return true
		}
	}
	return false
}
