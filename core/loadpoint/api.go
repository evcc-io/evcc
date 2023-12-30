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
	//
	// status
	//

	// GetStatus returns the charging status
	GetStatus() api.ChargeStatus

	//
	// settings
	//

	// Title returns the defined loadpoint title
	Title() string
	// GetPriority returns the priority
	GetPriority() int
	// SetPriority sets the priority
	SetPriority(int)
	// GetMinCurrent returns the min charging current
	GetMinCurrent() float64
	// SetMinCurrent sets the min charging current
	SetMinCurrent(float64)
	// GetMaxCurrent returns the max charging current
	GetMaxCurrent() float64
	// SetMaxCurrent sets the max charging current
	SetMaxCurrent(float64)

	// GetMode returns the charge mode
	GetMode() api.ChargeMode
	// SetMode sets the charge mode
	SetMode(api.ChargeMode)
	// GetPhases returns the enabled phases
	GetPhases() int
	// SetPhases sets the enabled phases
	SetPhases(int) error
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
	// EffectivePlanTime returns the effective plan time
	EffectivePlanTime() time.Time
	// EffectiveMinPower returns the min charging power for a single phase
	EffectiveMinPower() float64
	// EffectiveMaxPower returns the max charging power taking active phases into account
	EffectiveMaxPower() float64
	// PublishEffectiveValues publishes effective values for currently attached vehicle
	PublishEffectiveValues()

	//
	// plan
	//

	// GetPlanEnergy returns the charge plan energy
	GetPlanEnergy() (time.Time, float64)
	// SetPlanEnergy sets the charge plan energy
	SetPlanEnergy(time.Time, float64) error
	// GetPlanGoal returns the plan goal and if the goal is soc based
	GetPlanGoal() (float64, bool)
	// GetPlanRequiredDuration returns required duration of plan to reach the goal from current state
	GetPlanRequiredDuration(goal, maxPower float64) time.Duration
	// SocBasedPlanning determines if the planner is soc based
	SocBasedPlanning() bool
	// GetPlan creates a charging plan
	GetPlan(targetTime time.Time, requiredDuration time.Duration) (api.Rates, error)

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

	//
	// charge progress
	//

	// GetPlanActive returns the active state of the planner
	GetPlanActive() bool
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
