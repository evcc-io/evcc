package vehicle

import (
	"context"

	"github.com/evcc-io/evcc/api"
)

// TODO align phases with OnIdentify
type embed struct {
	_Title       string           `mapstructure:"title"` // TODO deprecated
	_Icon        string           `mapstructure:"-"`     // TODO deprecated
	Title_       string           `mapstructure:"-" json:"-"`
	Capacity_    float64          `mapstructure:"capacity"`
	Phases_      int              `mapstructure:"phases"`
	Identifiers_ []string         `mapstructure:"identifiers"`
	Features_    []api.Feature    `mapstructure:"features"`
	OnIdentify   api.ActionConfig `mapstructure:"onIdentify"`
}

func (v embed) withContext(ctx context.Context) *embed {
	if title := ctx.Value(api.ContextTitle); title != nil {
		v.Title_ = title.(string)
	}
	return &v
}

// GetTitle implements the api.Vehicle interface
func (v *embed) GetTitle() string {
	return v.Title_
}

// Capacity implements the api.Vehicle interface
func (v *embed) Capacity() float64 {
	return v.Capacity_
}

var _ api.PhaseDescriber = (*embed)(nil)

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

// var _ api.IconDescriber = (*embed)(nil)

// // Icon implements the api.IconDescriber interface
// func (v *embed) Icon() string {
// 	return v.Icon_
// }

var _ api.FeatureDescriber = (*embed)(nil)

// Features implements the api.FeatureDescriber interface
func (v *embed) Features() []api.Feature {
	return v.Features_
}
