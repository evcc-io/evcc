package vehicle

import (
	"github.com/evcc-io/evcc/api"
)

type embed struct {
	Title_            string           `mapstructure:"title"`
	Icon_             string           `mapstructure:"icon"`
	Capacity_         float64          `mapstructure:"capacity"`
	Phases_           int              `mapstructure:"phases"`
	DefaultTargetSoc_ int              `mapstructure:"targetSoc"`
	MinSoc_           int              `mapstructure:"minSoc"`
	Identifiers_      []string         `mapstructure:"identifiers"`
	Features_         []api.Feature    `mapstructure:"features"`
	OnIdentify        api.ActionConfig `mapstructure:"onIdentify"`
}

// Title implements the api.Vehicle interface
func (v *embed) Title() string {
	return v.Title_
}

// SetTitle implements the api.TitleSetter interface
func (v *embed) SetTitle(title string) {
	v.Title_ = title
}

// Capacity implements the api.Vehicle interface
func (v *embed) Capacity() float64 {
	return v.Capacity_
}

// Phases returns the phases used by the vehicle
func (v *embed) Phases() int {
	return v.Phases_
}

// DefaultTargetSoc implements the api.Vehicle interface
func (v *embed) DefaultTargetSoc() int {
	return v.DefaultTargetSoc_
}

// SetDefaultTargetSoc implements the api.Vehicle interface
func (v *embed) SetDefaultTargetSoc(soc int) {
	v.DefaultTargetSoc_ = soc
}

// MinSoc implements the api.Vehicle interface
func (v *embed) MinSoc() int {
	return v.MinSoc_
}

// SetMinSoc implements the api.Vehicle interface
func (v *embed) SetMinSoc(soc int) {
	v.MinSoc_ = soc
}

// Identifiers implements the api.Identifier interface
func (v *embed) Identifiers() []string {
	return v.Identifiers_
}

// OnIdentified returns the identify action
func (v *embed) OnIdentified() api.ActionConfig {
	return v.OnIdentify
}

var _ api.IconDescriber = (*embed)(nil)

// Icon implements the api.Vehicle interface
func (v *embed) Icon() string {
	return v.Icon_
}

var _ api.FeatureDescriber = (*embed)(nil)

// Features implements the api.FeatureDescriber interface
func (v *embed) Features() []api.Feature {
	return v.Features_
}
