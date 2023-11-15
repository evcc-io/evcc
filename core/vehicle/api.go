package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

//go:generate mockgen -package vehicle -destination mock.go -mock_names API=MockAPI github.com/evcc-io/evcc/core/vehicle API

type API interface {
	// Instance returns the vehicle instance
	Instance() api.Vehicle

	// Name returns the vehicle name
	Name() string

	// // GetMode returns the charge mode
	// GetMode() api.ChargeMode
	// // SetMode sets the charge mode
	// SetMode(api.ChargeMode)
	// // GetPhases returns the phases
	// GetPhases() int
	// // SetPhases sets the phases
	// SetPhases(phases int) error

	// // GetPriority returns the priority
	// GetPriority() int
	// // SetPriority sets the priority
	// SetPriority(priority int)

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

	// // GetMinCurrent returns the min charging current
	// GetMinCurrent() float64
	// // SetMinCurrent sets the min charging current
	// SetMinCurrent(float64)
	// // GetMaxCurrent returns the max charging current
	// GetMaxCurrent() float64
	// // SetMaxCurrent sets the max charging current
	// SetMaxCurrent(float64)
}
