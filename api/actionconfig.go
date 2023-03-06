package api

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/structs"
	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
)

// ActionConfig defines an action to take on event
type ActionConfig struct {
	Mode       *ChargeMode `mapstructure:"mode,omitempty"`       // Charge Mode
	MinCurrent *float64    `mapstructure:"minCurrent,omitempty"` // Minimum Current
	MaxCurrent *float64    `mapstructure:"maxCurrent,omitempty"` // Maximum Current
	MinSoc     *int        `mapstructure:"minSoc,omitempty"`     // Minimum Soc
	TargetSoc  *int        `mapstructure:"targetSoc,omitempty"`  // Target Soc
	Priority   *int        `mapstructure:"priority,omitempty"`   // Priority
}

// Merge merges all non-nil properties of the additional config into the base config.
// The receiver's config remains immutable.
func (a ActionConfig) Merge(m ActionConfig) ActionConfig {
	var res ActionConfig
	if err := copier.Copy(&res, a); err != nil {
		panic(err)
	}
	if err := mergo.MergeWithOverwrite(&res, m); err != nil {
		panic(err)
	}
	return res
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
