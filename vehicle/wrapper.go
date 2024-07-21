package vehicle

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	embed
	typ    string
	config map[string]interface{}
	err    error
}

// NewWrapper creates an offline Vehicle wrapper
func NewWrapper(name string, typ string, other map[string]interface{}, err error) api.Vehicle {
	var cc struct {
		embed `mapstructure:",squash"`
		Other map[string]interface{} `mapstructure:",remain"`
	}

	// try to decode vehicle-specific config and look for title attribute
	_ = util.DecodeOther(other, &cc)

	if cc.Title_ == "" {
		//lint:ignore SA1019 as Title is safe on ascii
		cc.Title_ = strings.Title(name)
	}

	v := &Wrapper{
		embed:  cc.embed,
		typ:    typ,
		config: cc.Other,
		err:    fmt.Errorf("vehicle not available: %w", err),
	}

	v.Features_ = append(v.Features_, api.Offline, api.Retryable)
	v.SetTitle(cc.Title_)

	return v
}

// Error returns the initialization error
func (v *Wrapper) Error() string {
	return v.err.Error()
}

// Error returns the initialization error
func (v *Wrapper) Config() (string, map[string]interface{}) {
	return v.typ, v.config
}

var _ api.Vehicle = (*Wrapper)(nil)

// SetTitle implements the api.TitleSetter interface
func (v *Wrapper) SetTitle(title string) {
	v.Title_ = title
}

var _ api.Battery = (*Wrapper)(nil)

// Soc implements the api.Battery interface
func (v *Wrapper) Soc() (float64, error) {
	return 0, v.err
}
