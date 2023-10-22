package loadpoint

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

//go:generate mockgen -package loadpoint -destination mock.go -mock_names API=MockAPI github.com/evcc-io/evcc/core/loadpoint API

// Controller gives access to loadpoint
type Controller interface {
	LoadpointControl(API)
}

// API is the external loadpoint API
type API interface {
	// Title returns the defined loadpoint title
	Title() string

	//
	// status
	//

	// GetStatus returns the charging status
	GetStatus() api.ChargeStatus

	//
	// effective settings
	//

	// GetEffectivePriority returns the effective priority
	GetEffectivePriority() int
	// GetEffectiveMinCurrent returns the effective min current
	GetEffectiveMinCurrent() float64
	// GetEffectiveMaxCurrent returns the effective max current
	GetEffectiveMaxCurrent() float64
	// GetEffectiveLimitSoc returns the effective session limit soc
	GetEffectiveLimitSoc() int

	//
	// settings
	//

	// GetPriority returns the priority
	GetPriority() int
	// SetPriority sets the priority
	SetPriority(int)

	// GetMode returns the charge mode
	GetMode() api.ChargeMode
	// SetMode sets the charge mode
	SetMode(api.ChargeMode)
	// GetPhases returns the enabled phases
	GetPhases() int
	// SetPhases sets the enabled phases
	SetPhases(int) error

	// GetSessionLimitSoc returns the session limit soc
	GetSessionLimitSoc() int
	// SetSessionLimitSoc sets the session limit soc
	SetSessionLimitSoc(soc int)
	// GetSessionLimitEnergy returns the session limit energy
	GetSessionLimitEnergy() float64
	// SetSessionLimitEnergy sets the session limit energy
	SetSessionLimitEnergy(energy float64)

	// GetPlanTime returns the plan time
	GetPlanTime() time.Time
	// GetPlanEnergy returns the charge plan energy
	GetPlanEnergy() float64
	// SetPlanEnergy sets the charge plan energy
	SetPlanEnergy(time.Time, float64) error
	// GetPlan creates a charging plan
	GetPlan(targetTime time.Time, maxPower float64) (time.Duration, api.Rates, error)

	// GetEnableThreshold gets the loadpoint enable threshold
	GetEnableThreshold() float64
	// SetEnableThreshold sets loadpoint enable threshold
	SetEnableThreshold(threshold float64)
	// GetDisableThreshold gets the loadpoint disable threshold
	GetDisableThreshold() float64
	// SetDisableThreshold sets loadpoint disable threshold
	SetDisableThreshold(threshold float64)

	// RemoteControl sets remote status demand
	RemoteControl(string, RemoteDemand)

	//
	// power and energy
	//

	// HasChargeMeter determines if a physical charge meter is attached
	HasChargeMeter() bool
	// GetChargePower returns the current charging power
	GetChargePower() float64
	// GetChargePowerFlexibility returns the flexible amount of current charging power
	GetChargePowerFlexibility() float64
	// TODO decide if needed
	// GetMinCurrent returns the min charging current
	GetMinCurrent() float64
	// SetMinCurrent sets the min charging current
	SetMinCurrent(float64)
	// GetMaxCurrent returns the max charging current
	GetMaxCurrent() float64
	// SetMaxCurrent sets the max charging current
	SetMaxCurrent(float64)
	// GetMinPower returns the min charging power for a single phase
	GetMinPower() float64
	// GetMaxPower returns the max charging power taking active phases into account
	GetMaxPower() float64

	//
	// charge progress
	//

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
