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
