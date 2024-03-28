package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

var _ API = (*dummy)(nil)

type dummy struct {
	api.Vehicle
}

func (v *dummy) Instance() api.Vehicle {
	return v.Vehicle
}

func (v *dummy) Name() string {
	return ""
}

// GetMode returns the charge mode
func (v *dummy) GetMode() api.ChargeMode {
	return api.ChargeMode("")
}

// SetMode sets the charge mode
func (v *dummy) SetMode(api.ChargeMode) {
}

// GetPhases returns the phases
func (v *dummy) GetPhases() int {
	return 0
}

// SetPhases sets the phases
func (v *dummy) SetPhases(phases int) {
}

// GetPriority returns the priority
func (v *dummy) GetPriority() int {
	return 0
}

// SetPriority sets the priority
func (v *dummy) SetPriority(priority int) {
}

// GetMinSoc returns the min soc
func (v *dummy) GetMinSoc() int {
	return 0
}

// SetMinSoc sets the min soc
func (v *dummy) SetMinSoc(soc int) {
}

// GetLimitSoc returns the limit soc
func (v *dummy) GetLimitSoc() int {
	return 0
}

// SetLimitSoc sets the limit soc
func (v *dummy) SetLimitSoc(soc int) {
}

// GetPlanSoc returns the charge plan soc
func (v *dummy) GetPlanSoc() (time.Time, int) {
	return time.Time{}, 0
}

// SetPlanSoc sets the charge plan soc
func (v *dummy) SetPlanSoc(ts time.Time, soc int) error {
	return nil
}

// GetMinCurrent returns the min current
func (v *dummy) GetMinCurrent() float64 {
	return 0
}

// SetMinCurrent sets the min current
func (v *dummy) SetMinCurrent(current float64) {
}

// GetMaxCurrent returns the max current
func (v *dummy) GetMaxCurrent() float64 {
	return 0
}

// SetMaxCurrent sets the max current
func (v *dummy) SetMaxCurrent(current float64) {
}
