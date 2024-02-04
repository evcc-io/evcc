package charger

import (
	"github.com/evcc-io/evcc/api"
)

type embed struct {
	Icon_     string        `mapstructure:"icon"`
	Features_ []api.Feature `mapstructure:"features"`
}

var _ api.PhaseDescriber = (*embed)(nil)

// Phases implements the api.PhasesDescriber interface
func (v *embed) Phases() int {
	// TODO make this configurable beyond switch sockets
	return 1
}

var _ api.IconDescriber = (*embed)(nil)

// Icon implements the api.IconDescriber interface
func (v *embed) Icon() string {
	return v.Icon_
}

var _ api.FeatureDescriber = (*embed)(nil)

// Features implements the api.FeatureDescriber interface
func (v *embed) Features() []api.Feature {
	return v.Features_
}
