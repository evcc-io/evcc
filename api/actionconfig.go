package api

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
)

// ActionConfig defines an action to take on event
type ActionConfig struct {
	Mode       ChargeMode `mapstructure:"mode,omitempty"`       // Charge Mode
	Priority   int        `mapstructure:"priority,omitempty"`   // Priority
	MinCurrent float64    `mapstructure:"minCurrent,omitempty"` // Minimum Current
	MaxCurrent float64    `mapstructure:"maxCurrent,omitempty"` // Maximum Current
}

// String implements Stringer and returns the ActionConfig as comma-separated key:value string
func (a ActionConfig) String() string {
	var s []string
	for _, f := range structs.Fields(a) {
		if !f.IsZero() {
			s = append(s, fmt.Sprintf("%s:%v", f.Name(), f.Value()))
		}
	}
	return strings.Join(s, ", ")
}

func (a ActionConfig) GetMode() (ChargeMode, bool) {
	return a.Mode, a.Mode != ""
}

func (a ActionConfig) GetMinCurrent() (float64, bool) {
	return a.MinCurrent, a.MinCurrent > 0
}

func (a ActionConfig) GetMaxCurrent() (float64, bool) {
	return a.MaxCurrent, a.MaxCurrent > 0
}

func (a ActionConfig) GetPriority() (int, bool) {
	return a.Priority, a.Priority > 0
}
