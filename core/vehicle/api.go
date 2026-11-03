package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

//go:generate go tool mockgen -package vehicle -destination mock.go -mock_names API=MockAPI github.com/evcc-io/evcc/core/vehicle API

type API interface {
	// Instance returns the vehicle instance
	Instance() api.Vehicle

	// Name returns the vehicle name
	Name() string

	// GetMode returns the charge mode
	GetMode() api.ChargeMode
	// SetMode sets the charge mode
	SetMode(api.ChargeMode)
	// GetAlwaysCharge returns the always charge state applied on identification, empty if unset
	GetAlwaysCharge() api.AlwaysCharge
	// SetAlwaysCharge sets the always charge state applied on identification, empty clears it
	SetAlwaysCharge(api.AlwaysCharge)

	// GetMinSoc returns the min soc
	GetMinSoc() int
	// SetMinSoc sets the min soc
	SetMinSoc(soc int)
	// GetLimitSoc returns the limit soc
	GetLimitSoc() int
	// SetLimitSoc sets the limit soc
	SetLimitSoc(soc int)

	// GetPlanSoc returns the charge plan soc
	GetPlanSoc() (time.Time, int)
	// SetPlanSoc sets the charge plan time and soc
	SetPlanSoc(time.Time, int) error

	// GetRepeatingPlans returns every repeating plan
	GetRepeatingPlans() []api.RepeatingPlan
	// SetRepeatingPlans stores every repeating plan
	SetRepeatingPlans([]api.RepeatingPlan) error

	// GetPlanStrategy returns the plan strategy
	GetPlanStrategy() api.PlanStrategy
	// SetPlanStrategy sets the plan strategy
	SetPlanStrategy(api.PlanStrategy) error

	// GetFullCharge returns the periodic full charge thresholds
	GetFullCharge() api.FullCharge
	// SetFullCharge sets the periodic full charge thresholds
	SetFullCharge(api.FullCharge) error
	// GetFullChargeTime returns the time of the last full charge
	GetFullChargeTime() time.Time
	// SetFullChargeTime sets the time of the last full charge and resets the energy since
	SetFullChargeTime(time.Time)
	// GetFullChargeEnergy returns the energy charged since the last full charge
	GetFullChargeEnergy() float64
	// AddFullChargeEnergy adds charged energy since the last full charge
	AddFullChargeEnergy(float64)
}
