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
	MinSoc_    *int        `mapstructure:"minSoc,omitempty"`     // Minimum Soc (deprecated)
	TargetSoc_ *int        `mapstructure:"targetSoc,omitempty"`  // Target Soc (deprecated)
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
