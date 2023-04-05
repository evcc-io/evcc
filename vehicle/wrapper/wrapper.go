package wrapper

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	err       error
	title     string
	icon      string
	phases    int
	capacity  float64
	Features_ []api.Feature
}

// New creates a new Vehicle
func New(name string, other map[string]interface{}, err error) api.Vehicle {
	var cc struct {
		Title    string
		Icon     string
		Phases   int
		Capacity float64
		Other    map[string]interface{} `mapstructure:",remain"`
	}

	// try to decode vehicle-specific config and look for title attribute
	_ = util.DecodeOther(other, &cc)

	if cc.Title == "" {
		//lint:ignore SA1019 as Title is safe on ascii
		cc.Title = strings.Title(name)
	}

	v := &Wrapper{
		err:       fmt.Errorf("vehicle not available: %w", err),
		title:     fmt.Sprintf("%s (offline)", cc.Title),
		icon:      cc.Icon,
		phases:    cc.Phases,
		capacity:  cc.Capacity,
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
	return v.icon
}

// Capacity implements the api.Vehicle interface
func (v *Wrapper) Capacity() float64 {
	return v.capacity
}

// Phases implements the api.Vehicle interface
func (v *Wrapper) Phases() int {
	return v.phases
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
