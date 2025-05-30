package loadpoint

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

//go:generate go tool mockgen -package loadpoint -destination mock.go -mock_names API=MockAPI github.com/evcc-io/evcc/core/loadpoint API

// Controller gives access to loadpoint
type Controller interface {
	LoadpointControl(API)
}

// API is the external loadpoint API
type API interface {
	//
	// status
	//

	// GetStatus returns the charging status
	GetStatus() api.ChargeStatus

	//
	// references
	//

	// GetChargerRef returns the loadpoint charger
	GetChargerRef() string
	// SetChargerRef sets the loadpoint charger
	SetChargerRef(string)
	// GetMeterRef returns the loadpoint meter
	GetMeterRef() string
	// SetMeterRef sets the loadpoint meter
	SetMeterRef(string)
	// GetCircuitRef returns the loadpoint circuit
	GetCircuitRef() string
	// SetCircuitRef sets the loadpoint circuit
	SetCircuitRef(string)
	// GetCircuit returns the loadpoint circuit
	GetCircuit() api.Circuit
	// GetDefaultVehicleRef returns the loadpoint default vehicle
	GetDefaultVehicleRef() string
	// SetDefaultVehicleRef sets the loadpoint default vehicle
	SetDefaultVehicleRef(string)

	//
	// settings
	//

	// GetTitle returns the loadpoint title
	GetTitle() string
	// SetTitle sets the loadpoint title
	SetTitle(string)
	// GetPriority returns the priority
	GetPriority() int
	// SetPriority sets the priority
	SetPriority(int)
	// GetMinCurrent returns the min charging current
	GetMinCurrent() float64
	// SetMinCurrent sets the min charging current
	SetMinCurrent(float64) error
	// GetMaxCurrent returns the max charging current
	GetMaxCurrent() float64
	// SetMaxCurrent sets the max charging current
	SetMaxCurrent(float64) error

	// GetMode returns the current charge mode
	GetMode() api.ChargeMode
	// SetMode sets the charge mode
	SetMode(api.ChargeMode)
	// GetDefaultMode returns the default charge mode (for reset)
	GetDefaultMode() api.ChargeMode
	// SetDefaultMode sets the default charge mode (for reset)
	SetDefaultMode(api.ChargeMode)
	// GetPhases returns the enabled phases
	GetPhases() int
	// GetPhasesConfigured returns the configured phases
	GetPhasesConfigured() int
	// SetPhasesConfigured sets the configured phases
	SetPhasesConfigured(int) error
	// ActivePhases returns the active phases for the current vehicle
	ActivePhases() int

	// GetLimitSoc returns the session limit soc
	GetLimitSoc() int
	// SetLimitSoc sets the session limit soc
	SetLimitSoc(soc int)
	// GetLimitEnergy returns the session limit energy
	GetLimitEnergy() float64
	// SetLimitEnergy sets the session limit energy
	SetLimitEnergy(energy float64)

	//
	// effective values
	//

	// EffectivePriority returns the effective priority
	EffectivePriority() int
	// EffectivePlanId returns the effective plan id
	EffectivePlanId() int
	// EffectivePlanTime returns the effective plan time
	EffectivePlanTime() time.Time
	// EffectiveMinPower returns the min charging power for the minimum active phases
	EffectiveMinPower() float64
	// EffectiveMaxPower returns the max charging power taking active phases into account
	EffectiveMaxPower() float64
	// PublishEffectiveValues publishes effective values for currently attached vehicle
	PublishEffectiveValues()

	//
	// plan
	//

	// GetPlanEnergy returns the charge plan energy
	GetPlanEnergy() (time.Time, time.Duration, float64)
	// SetPlanEnergy sets the charge plan energy
	SetPlanEnergy(time.Time, time.Duration, float64) error
	// GetPlanGoal returns the plan goal, precondition duration and if the goal is soc based
	GetPlanGoal() (float64, bool)
	// GetPlanRequiredDuration returns required duration of plan to reach the goal from current state
	GetPlanRequiredDuration(goal, maxPower float64) time.Duration
	// GetPlanPreCondDuration returns the precondition duration
	GetPlanPreCondDuration() time.Duration
	// SocBasedPlanning determines if the planner is soc based
	SocBasedPlanning() bool
	// GetPlan creates a charging plan
	GetPlan(targetTime time.Time, requiredDuration, precondition time.Duration) api.Rates

	// GetSocConfig returns the soc poll settings
	GetSocConfig() SocConfig
	// SetSocConfig sets the soc poll settings
	SetSocConfig(soc SocConfig)

	// GetThresholds returns the PV mode threshold settings
	GetThresholds() ThresholdsConfig
	// SetThresholds sets the PV mode threshold settings
	SetThresholds(thresholds ThresholdsConfig)
	// GetEnableThreshold gets the loadpoint enable threshold
	GetEnableThreshold() float64
	// SetEnableThreshold sets loadpoint enable threshold
	SetEnableThreshold(threshold float64)
	// GetDisableThreshold gets the loadpoint disable threshold
	GetDisableThreshold() float64
	// SetDisableThreshold sets loadpoint disable threshold
	SetDisableThreshold(threshold float64)

	// GetEnableDelay gets the loadpoint enable delay
	GetEnableDelay() time.Duration
	// SetEnableDelay sets loadpoint enable delay
	SetEnableDelay(delay time.Duration)
	// GetDisableDelay gets the loadpoint disable delay
	GetDisableDelay() time.Duration
	// SetDisableDelay sets loadpoint disable delay
	SetDisableDelay(delay time.Duration)

	// GetBatteryBoost returns the battery boost
	GetBatteryBoost() int
	// SetBatteryBoost sets the battery boost
	SetBatteryBoost(enable bool) error

	// RemoteControl sets remote status demand
	RemoteControl(string, RemoteDemand)

	//
	// smart grid charging
	//

	// GetSmartChargingActive determines if smart charging is active
	GetSmartCostLimit() *float64
	// SetSmartCostLimit sets the smart cost limit
	SetSmartCostLimit(limit *float64)

	//
	// power and energy
	//

	// HasChargeMeter determines if a physical charge meter is attached
	HasChargeMeter() bool
	// GetChargePower returns the current charging power
	GetChargePower() float64
	// GetChargePowerFlexibility returns the flexible amount of current charging power
	GetChargePowerFlexibility(rates api.Rates) float64
	// GetMaxPhaseCurrent returns max phase current
	GetMaxPhaseCurrent() float64

	//
	// charge progress
	//

	// IsFastChargingActive indicates if fast charging with maximum power is active
	IsFastChargingActive() bool
	// GetRemainingDuration is the estimated remaining charging duration
	GetRemainingDuration() time.Duration
	// GetRemainingEnergy is the remaining charge energy in Wh
	GetRemainingEnergy() float64

	//
	// vehicles
	//

	// GetVehicle gets the active vehicle
	GetVehicle() api.Vehicle
	// SetVehicle sets the active vehicle
	SetVehicle(vehicle api.Vehicle)
	// StartVehicleDetection allows triggering vehicle detection for debugging purposes
	StartVehicleDetection()
}
