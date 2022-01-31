package vehicle

import "github.com/evcc-io/evcc/api"

type embed struct {
	Title_       string           `mapstructure:"title"`
	Capacity_    int64            `mapstructure:"capacity"`
	Phases_      int              `mapstructure:"phases"`
	Identifiers_ []string         `mapstructure:"identifiers"`
	OnIdentify   api.ActionConfig `mapstructure:"onIdentify"`
}

// Title implements the api.Vehicle interface
func (v *embed) Title() string {
	return v.Title_
}

// Capacity implements the api.Vehicle interface
func (v *embed) Capacity() int64 {
	return v.Capacity_
}

// Phases returns the phases used by the vehicle
func (v *embed) Phases() int {
	return v.Phases_
}

// Identifiers implements the api.Identifier interface
func (v *embed) Identifiers() []string {
	return v.Identifiers_
}

// OnIdentified returns the identify action
func (v *embed) OnIdentified() api.ActionConfig {
	return v.OnIdentify
}
