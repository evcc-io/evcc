package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/core/wrapper"
)

var _ loadpoint.API = (*Loadpoint)(nil)

func (lp *Loadpoint) isConfigurable() bool {
	_, ok := lp.settings.(*settings.ConfigSettings)
	return ok
}

// GetChargerRef returns the loadpoint charger
func (lp *Loadpoint) GetChargerRef() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.ChargerRef
}

// SetChargerRef sets the loadpoint charger
func (lp *Loadpoint) SetChargerRef(ref string) {
	if !lp.isConfigurable() {
		lp.log.ERROR.Println("cannot set charger ref: not configurable")
		return
	}

	lp.Lock()
	defer lp.Unlock()
	lp.ChargerRef = ref
	lp.settings.SetString(keys.Charger, ref)
}

// GetMeter returns the loadpoint meter
func (lp *Loadpoint) GetMeterRef() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.MeterRef
}

// SetMeter sets the loadpoint meter
func (lp *Loadpoint) SetMeterRef(ref string) {
	if !lp.isConfigurable() {
		lp.log.ERROR.Println("cannot set meter ref: not configurable")
		return
	}

	lp.Lock()
	defer lp.Unlock()
	lp.MeterRef = ref
	lp.settings.SetString(keys.Meter, ref)
}

// GetCircuitName returns the loadpoint circuit
func (lp *Loadpoint) GetCircuitRef() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.CircuitRef
}

// SetCircuitRef sets the loadpoint circuit
func (lp *Loadpoint) SetCircuitRef(ref string) {
	if !lp.isConfigurable() {
		lp.log.ERROR.Println("cannot set circuit ref: not configurable")
		return
	}

	lp.log.DEBUG.Println("set circuit ref:", ref)

	lp.Lock()
	defer lp.Unlock()
	lp.CircuitRef = ref
	lp.settings.SetString(keys.Circuit, ref)
}

// GetDefaultVehicleRef returns the loadpoint default vehicle
func (lp *Loadpoint) GetDefaultVehicleRef() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.VehicleRef
}

// SetDefaultVehicleRef returns the loadpoint default vehicle
func (lp *Loadpoint) SetDefaultVehicleRef(ref string) {
	if !lp.isConfigurable() {
		lp.log.ERROR.Println("cannot set default vehicle ref: not configurable")
		return
	}

	lp.log.DEBUG.Println("set default vehicle ref:", ref)

	lp.Lock()
	defer lp.Unlock()
	lp.VehicleRef = ref
	lp.settings.SetString(keys.DefaultVehicle, ref)
}

// GetTitle returns the loadpoint title
func (lp *Loadpoint) GetTitle() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.title
}

// SetTitle sets the loadpoint title
func (lp *Loadpoint) SetTitle(title string) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set title:", title)

	if title != lp.title {
		lp.setTitle(title)
	}
}

// setTitle sets the loadpoint title (no mutex)
func (lp *Loadpoint) setTitle(title string) {
	lp.title = title
	lp.publish(keys.Title, lp.title)
	lp.settings.SetString(keys.Title, lp.title)
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
	return lp.mode
}

// setMode sets loadpoint charge mode (no mutex)
func (lp *Loadpoint) setMode(mode api.ChargeMode) {
	lp.mode = mode
	lp.publish(keys.Mode, mode)
	lp.settings.SetString(keys.Mode, string(mode))
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
	if lp.mode != mode {
		lp.setMode(mode)

		lp.batteryBoost = boostDisabled
		lp.publish(keys.BatteryBoost, false)

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

// GetDefaultMode returns the default charge mode
func (lp *Loadpoint) GetDefaultMode() api.ChargeMode {
	lp.RLock()
	defer lp.RUnlock()
	return lp.DefaultMode
}

// SetDefaultMode sets the default charge mode
func (lp *Loadpoint) SetDefaultMode(mode api.ChargeMode) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set default mode:", mode)

	if lp.DefaultMode != mode {
		lp.DefaultMode = mode
		lp.settings.SetString(keys.DefaultMode, string(mode))
	}
}

// GetChargedEnergy returns session charge energy in Wh
func (lp *Loadpoint) GetChargedEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getChargedEnergy()
}

// getChargedEnergy returns session charge energy in Wh
func (lp *Loadpoint) getChargedEnergy() float64 {
	return lp.energyMetrics.TotalWh()
}

// GetPriority returns the loadpoint priority
func (lp *Loadpoint) GetPriority() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.priority
}

// setPriority sets the loadpoint priority (no mutex)
func (lp *Loadpoint) setPriority(prio int) {
	lp.priority = prio
	lp.publish(keys.Priority, lp.priority)
	lp.settings.SetInt(keys.Priority, int64(lp.priority))
}

// SetPriority sets the loadpoint priority
func (lp *Loadpoint) SetPriority(prio int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set priority:", prio)
	if lp.priority != prio {
		lp.setPriority(prio)
	}
}

// GetPhases returns the enabled phases
func (lp *Loadpoint) GetPhases() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.phases
}

// GetPhasesConfigured returns the configured phases
func (lp *Loadpoint) GetPhasesConfigured() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.phasesConfigured
}

// SetPhasesConfigured sets the configured phases
func (lp *Loadpoint) SetPhasesConfigured(phases int) error {
	// limit auto mode (phases=0) to scalable charger
	if !lp.hasPhaseSwitching() && phases == 0 {
		return fmt.Errorf("charger does not support phase switching")
	}

	if phases != 0 && phases != 1 && phases != 3 {
		return fmt.Errorf("invalid number of phases: %d", phases)
	}

	// set new default
	lp.log.DEBUG.Println("set phases:", phases)

	lp.Lock()
	lp.setPhasesConfigured(phases)
	lp.Unlock()

	lp.requestUpdate()

	return nil
}

// GetLimitSoc returns the session limit soc
func (lp *Loadpoint) GetLimitSoc() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.limitSoc
}

// setLimitSoc sets the session limit soc (no mutex)
func (lp *Loadpoint) setLimitSoc(soc int) {
	lp.limitSoc = soc
	lp.publish(keys.LimitSoc, soc)
	lp.settings.SetInt(keys.LimitSoc, int64(soc))
}

// SetLimitSoc sets the session soc limit
func (lp *Loadpoint) SetLimitSoc(soc int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set session soc limit:", soc)

	// apply immediately
	if lp.limitSoc != soc {
		lp.setLimitSoc(soc)
		lp.requestUpdate()
	}
}

// GetLimitEnergy returns the session limit energy
func (lp *Loadpoint) GetLimitEnergy() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getLimitEnergy()
}

// getLimitEnergy returns the session limit energy
func (lp *Loadpoint) getLimitEnergy() float64 {
	return lp.limitEnergy
}

// setLimitEnergy sets the session limit energy (no mutex)
func (lp *Loadpoint) setLimitEnergy(energy float64) {
	lp.limitEnergy = energy
	lp.publish(keys.LimitEnergy, energy)
	lp.settings.SetFloat(keys.LimitEnergy, energy)
}

// SetLimitEnergy sets the session energy limit
func (lp *Loadpoint) SetLimitEnergy(energy float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set session energy limit:", energy)

	// apply immediately
	if lp.limitEnergy != energy {
		lp.setLimitEnergy(energy)
		lp.requestUpdate()
	}
}

// GetPlanEnergy returns plan target energy
func (lp *Loadpoint) GetPlanEnergy() (time.Time, time.Duration, float64) {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getPlanEnergy()
}

// getPlanEnergy returns plan target energy
func (lp *Loadpoint) getPlanEnergy() (time.Time, time.Duration, float64) {
	return lp.planTime, lp.planPrecondition, lp.planEnergy
}

// setPlanEnergy sets plan target energy (no mutex)
func (lp *Loadpoint) setPlanEnergy(finishAt time.Time, precondition time.Duration, energy float64) {
	lp.planEnergy = energy
	lp.publish(keys.PlanEnergy, energy)
	lp.settings.SetFloat(keys.PlanEnergy, energy)

	// remove plan
	if energy == 0 {
		finishAt = time.Time{}
		precondition = 0
	}

	lp.planTime = finishAt
	lp.planPrecondition = precondition
	lp.publish(keys.PlanTime, finishAt)
	lp.publish(keys.PlanPrecondition, precondition)
	lp.settings.SetTime(keys.PlanTime, finishAt)
	lp.settings.SetInt(keys.PlanPrecondition, int64(precondition.Seconds()))

	if finishAt.IsZero() {
		lp.setPlanActive(false)
	}
}

// SetPlanEnergy sets plan target energy
func (lp *Loadpoint) SetPlanEnergy(finishAt time.Time, precondition time.Duration, energy float64) error {
	lp.Lock()
	defer lp.Unlock()

	if !finishAt.IsZero() && finishAt.Before(lp.clock.Now()) {
		return errors.New("timestamp is in the past")
	}

	lp.log.DEBUG.Printf("set plan energy: %.3gkWh @ %v", energy, finishAt.Round(time.Second).Local())

	// apply immediately
	if lp.planEnergy != energy || lp.planPrecondition != precondition || !lp.planTime.Equal(finishAt) {
		lp.setPlanEnergy(finishAt, precondition, energy)
		lp.requestUpdate()
	}

	return nil
}

// GetSoc returns the PV mode threshold settings
func (lp *Loadpoint) GetSocConfig() loadpoint.SocConfig {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Soc
}

func (lp *Loadpoint) setSocConfig(soc loadpoint.SocConfig) {
	lp.Soc = soc
	lp.settings.SetJson(keys.Soc, soc)
	lp.requestUpdate()
}

// SetSoc sets the PV mode threshold settings
func (lp *Loadpoint) SetSocConfig(soc loadpoint.SocConfig) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Printf("set soc config: %+v", soc)

	// apply immediately
	lp.setSocConfig(soc)
}

// GetThresholds returns the PV mode threshold settings
func (lp *Loadpoint) GetThresholds() loadpoint.ThresholdsConfig {
	lp.RLock()
	defer lp.RUnlock()
	return loadpoint.ThresholdsConfig{
		Enable:  lp.Enable,
		Disable: lp.Disable,
	}
}

func (lp *Loadpoint) setThresholds(thresholds loadpoint.ThresholdsConfig) {
	lp.Enable = thresholds.Enable
	lp.Disable = thresholds.Disable
	lp.publish(keys.EnableThreshold, lp.Enable.Threshold)
	lp.publish(keys.DisableThreshold, lp.Disable.Threshold)
	lp.settings.SetJson(keys.Thresholds, thresholds)
	lp.requestUpdate()
}

// SetThresholds sets the PV mode threshold settings
func (lp *Loadpoint) SetThresholds(thresholds loadpoint.ThresholdsConfig) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Printf("set thresholds: %+v", thresholds)

	// apply immediately
	lp.setThresholds(thresholds)
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
		// TODO reduce APIs
		lp.setThresholds(loadpoint.ThresholdsConfig{
			Enable:  lp.Enable,
			Disable: lp.Disable,
		})
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
		// TODO reduce APIs
		lp.setThresholds(loadpoint.ThresholdsConfig{
			Enable:  lp.Enable,
			Disable: lp.Disable,
		})
	}
}

// GetEnableDelay gets the loadpoint enable delay
func (lp *Loadpoint) GetEnableDelay() time.Duration {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Enable.Delay
}

// SetEnableDelay sets loadpoint enable delay
func (lp *Loadpoint) SetEnableDelay(delay time.Duration) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set enable delay:", delay)

	if lp.Enable.Delay != delay {
		lp.Enable.Delay = delay
		lp.publish(keys.EnableDelay, delay)
	}
}

// GetDisableDelay gets the loadpoint enable delay
func (lp *Loadpoint) GetDisableDelay() time.Duration {
	lp.RLock()
	defer lp.RUnlock()
	return lp.Disable.Delay
}

// SetDisableDelay sets loadpoint disable delay
func (lp *Loadpoint) SetDisableDelay(delay time.Duration) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set disable delay:", delay)

	if lp.Disable.Delay != delay {
		lp.Disable.Delay = delay
		lp.publish(keys.DisableDelay, delay)
	}
}

// GetBatteryBoost returns the battery boost
func (lp *Loadpoint) GetBatteryBoost() int {
	lp.RLock()
	defer lp.RUnlock()
	return lp.batteryBoost
}

// setBatteryBoost returns the battery boost
func (lp *Loadpoint) setBatteryBoost(boost int) {
	lp.Lock()
	defer lp.Unlock()
	lp.batteryBoost = boost
}

// SetBatteryBoost sets the battery boost
func (lp *Loadpoint) SetBatteryBoost(enable bool) error {
	lp.Lock()
	defer lp.Unlock()

	if enable && lp.mode != api.ModePV && lp.mode != api.ModeMinPV {
		return errors.New("battery boost is only available in PV modes")
	}

	lp.log.DEBUG.Println("set battery boost:", enable)

	if enable != (lp.batteryBoost != boostDisabled) {
		lp.publish(keys.BatteryBoost, enable)

		lp.batteryBoost = boostDisabled
		if enable {
			lp.batteryBoost = boostStart
			lp.requestUpdate()
		}
	}

	return nil
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
func (lp *Loadpoint) GetChargePowerFlexibility(rates api.Rates) float64 {
	mode := lp.GetMode()
	if mode == api.ModeNow || !lp.charging() || lp.minSocNotReached() || lp.smartCostActive(rates) {
		return 0
	}

	if mode == api.ModePV {
		return lp.GetChargePower()
	}

	// MinPV mode
	return max(0, lp.GetChargePower()-lp.EffectiveMinPower())
}

// GetMaxPhaseCurrent returns the current charge power
func (lp *Loadpoint) GetMaxPhaseCurrent() float64 {
	lp.RLock()
	defer lp.RUnlock()
	if lp.chargeCurrents == nil {
		return lp.offeredCurrent
	}
	return max(lp.chargeCurrents[0], lp.chargeCurrents[1], lp.chargeCurrents[2])
}

// GetMinCurrent returns the min loadpoint current
func (lp *Loadpoint) GetMinCurrent() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getMinCurrent()
}

// getMinCurrent returns the max loadpoint current
func (lp *Loadpoint) getMinCurrent() float64 {
	return lp.minCurrent
}

// setMinCurrent sets the min loadpoint current (no mutex)
func (lp *Loadpoint) setMinCurrent(current float64) {
	lp.minCurrent = current
	lp.publish(keys.MinCurrent, lp.minCurrent)
	lp.settings.SetFloat(keys.MinCurrent, lp.minCurrent)
}

// SetMinCurrent sets the min loadpoint current
func (lp *Loadpoint) SetMinCurrent(current float64) error {
	lp.Lock()
	defer lp.Unlock()

	if current > lp.maxCurrent {
		return errors.New("min current must be smaller or equal than max current")
	}

	lp.log.DEBUG.Println("set min current:", current)
	if current != lp.minCurrent {
		lp.setMinCurrent(current)
	}

	return nil
}

// GetMaxCurrent returns the max loadpoint current
func (lp *Loadpoint) GetMaxCurrent() float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.getMaxCurrent()
}

// getMaxCurrent returns the max loadpoint current
func (lp *Loadpoint) getMaxCurrent() float64 {
	return lp.maxCurrent
}

// setMaxCurrent sets the max loadpoint current
func (lp *Loadpoint) setMaxCurrent(current float64) {
	lp.maxCurrent = current
	lp.publish(keys.MaxCurrent, lp.maxCurrent)
	lp.settings.SetFloat(keys.MaxCurrent, lp.maxCurrent)
}

// SetMaxCurrent sets the max loadpoint current
func (lp *Loadpoint) SetMaxCurrent(current float64) error {
	lp.Lock()
	defer lp.Unlock()

	if current < lp.minCurrent {
		return errors.New("max current must be greater or equal than min current")
	}

	lp.log.DEBUG.Println("set max current:", current)
	if current != lp.maxCurrent {
		lp.setMaxCurrent(current)
	}

	return nil
}

// IsFastChargingActive indicates if fast charging with maximum power is active
func (lp *Loadpoint) IsFastChargingActive() bool {
	lp.RLock()
	defer lp.RUnlock()

	return lp.mode == api.ModeNow || lp.planActive || lp.minSocNotReached()
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
	lp.vmu.RLock()
	defer lp.vmu.RUnlock()
	return lp.vehicle
}

// SetVehicle sets the active vehicle
func (lp *Loadpoint) SetVehicle(vehicle api.Vehicle) {
	// set desired vehicle (protected by lock, no locking here)
	lp.setActiveVehicle(vehicle)

	lp.vmu.Lock()
	defer lp.vmu.Unlock()

	// disable auto-detect
	lp.stopVehicleDetection()
}

// StartVehicleDetection allows triggering vehicle detection for debugging purposes
func (lp *Loadpoint) StartVehicleDetection() {
	// reset vehicle
	lp.setActiveVehicle(nil)

	lp.vmu.Lock()
	defer lp.vmu.Unlock()

	// start auto-detect
	lp.startVehicleDetection()
}

// GetSmartCostLimit gets the smart cost limit
func (lp *Loadpoint) GetSmartCostLimit() *float64 {
	lp.RLock()
	defer lp.RUnlock()
	return lp.smartCostLimit
}

// SetSmartCostLimit sets the smart cost limit
func (lp *Loadpoint) SetSmartCostLimit(val *float64) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.DEBUG.Println("set smart cost limit:", printPtr("%.1f", val))

	if !ptrValueEqual(lp.smartCostLimit, val) {
		lp.smartCostLimit = val

		lp.settings.SetFloatPtr(keys.SmartCostLimit, val)
		lp.publish(keys.SmartCostLimit, val)
	}
}

// GetCircuit returns the assigned circuit
func (lp *Loadpoint) GetCircuit() api.Circuit {
	lp.RLock()
	defer lp.RUnlock()

	// return untyped nil
	if lp.circuit == nil {
		return nil
	}

	return lp.circuit
}
