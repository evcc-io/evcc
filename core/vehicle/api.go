package vehicle

import "github.com/evcc-io/evcc/api"

type API interface {
	// GetMode returns the charge mode
	GetMode() api.ChargeMode
	// SetMode sets the charge mode
	SetMode(api.ChargeMode)
	// GetPhases returns the limit soc
	GetPhases() int
	// SetPhases sets the limit soc
	SetPhases(phases int) error

	// GetPriority returns the priority
	GetPriority() int
	// SetPriority sets the priority
	SetPriority(priority int)

	// GetMinSoc returns the min soc
	GetMinSoc() int
	// SetMinSoc sets the min soc
	SetMinSoc(soc int)
	// GetLimitSoc returns the limit soc
	GetLimitSoc() int
	// SetLimitSoc sets the limit soc
	SetLimitSoc(soc int)

	// GetMinCurrent returns the min charging current
	GetMinCurrent() float64
	// SetMinCurrent sets the min charging current
	SetMinCurrent(float64)
	// GetMaxCurrent returns the max charging current
	GetMaxCurrent() float64
	// SetMaxCurrent sets the max charging current
	SetMaxCurrent(float64)
}
