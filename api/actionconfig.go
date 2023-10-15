package api

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/structs"
)

// ActionConfig defines an action to take on event
type ActionConfig struct {
	Mode       *ChargeMode `mapstructure:"mode,omitempty"`       // Charge Mode
	MinCurrent *float64    `mapstructure:"minCurrent,omitempty"` // Minimum Current
	MaxCurrent *float64    `mapstructure:"maxCurrent,omitempty"` // Maximum Current
	MinSoc     *int        `mapstructure:"minSoc,omitempty"`     // Minimum Soc (vehicle only)
	LimitSoc   *int        `mapstructure:"limitSoc,omitempty"`   // Limit Soc
	Priority   *int        `mapstructure:"priority,omitempty"`   // Priority
}

// String implements Stringer and returns the ActionConfig as comma-separated key:value string
func (a ActionConfig) String() string {
	var s []string
	for k, v := range structs.Map(a) {
		val := reflect.ValueOf(v)
		if v != nil && !val.IsNil() {
			s = append(s, fmt.Sprintf("%s:%v", k, val.Elem()))
		}
	}
	return strings.Join(s, ", ")
}

func (a ActionConfig) GetMode() (ChargeMode, error) {
	if a.Mode == nil {
		return "", ErrNotAvailable
	}
	return *a.Mode, nil
}

func (a ActionConfig) GetMinCurrent() (float64, error) {
	if a.MinCurrent == nil {
		return 0, ErrNotAvailable
	}
	return *a.MinCurrent, nil
}

func (a ActionConfig) GetMaxCurrent() (float64, error) {
	if a.MaxCurrent == nil {
		return 0, ErrNotAvailable
	}
	return *a.MaxCurrent, nil
}

func (a ActionConfig) GetMinSoc() (int, error) {
	if a.MinSoc == nil {
		return 0, ErrNotAvailable
	}
	return *a.MinSoc, nil
}

func (a ActionConfig) GetLimitSoc() (int, error) {
	if a.LimitSoc == nil {
		return 0, ErrNotAvailable
	}
	return *a.LimitSoc, nil
}

func (a ActionConfig) GetPriority() (int, error) {
	if a.Priority == nil {
		return 0, ErrNotAvailable
	}
	return *a.Priority, nil
}
