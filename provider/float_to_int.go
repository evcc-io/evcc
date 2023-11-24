package provider

import (
	"github.com/evcc-io/evcc/util"
)

type floatToIntProvider struct {
	Set Config
}

func init() {
	registry.Add("float2int", NewFloatToIntFromConfig)
}

// NewFloatToIntFromConfig creates type conversion provider
func NewFloatToIntFromConfig(other map[string]interface{}) (Provider, error) {
	var cc floatToIntProvider

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}

var _ SetFloatProvider = (*floatToIntProvider)(nil)

func (o *floatToIntProvider) FloatSetter(param string) (func(float64) error, error) {
	set, err := NewIntSetterFromConfig(param, o.Set)

	return func(val float64) error {
		return set(int64(val))
	}, err
}
