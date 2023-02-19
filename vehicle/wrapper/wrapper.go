package wrapper

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
)

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	err       error
	title     string
	Features_ []api.Feature
}

// New creates a new Vehicle
func New(err error) api.Vehicle {
	v := &Wrapper{
		err:       fmt.Errorf("vehicle not available: %w", err),
		title:     "unavailable",
		Features_: []api.Feature{api.Offline},
	}

	return v
}

var _ api.Vehicle = (*Wrapper)(nil)

// Title implements the api.Vehicle interface
func (v *Wrapper) Title() string {
	return v.title
}

// SetTitle implements the api.TitleSetter interface
func (v *Wrapper) SetTitle(title string) {
	v.title = fmt.Sprintf("%s (unavailable)", title)
}

// Icon implements the api.Vehicle interface
func (v *Wrapper) Icon() string {
	return ""
}

// Capacity implements the api.Vehicle interface
func (v *Wrapper) Capacity() float64 {
	return 0
}

// Phases implements the api.Vehicle interface
func (v *Wrapper) Phases() int {
	return 0
}

// Identifiers implements the api.Vehicle interface
func (v *Wrapper) Identifiers() []string {
	return nil
}

// OnIdentified implements the api.Vehicle interface
func (v *Wrapper) OnIdentified() api.ActionConfig {
	return api.ActionConfig{}
}

var _ api.FeatureDescriber = (*Wrapper)(nil)

// Features implements the api.FeatureDescriber interface
func (v *Wrapper) Features() []api.Feature {
	return []api.Feature{api.Offline}
}

var _ api.Battery = (*Wrapper)(nil)

// Soc implements the api.Battery interface
func (v *Wrapper) Soc() (float64, error) {
	return 0, v.err
}
