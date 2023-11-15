package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type floatToIntProvider struct {
	set func(int64) error
}

func init() {
	registry.Add("float2int", NewFloatToIntFromConfig)
}

// NewFloatToIntFromConfig creates type conversion provider
func NewFloatToIntFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Param string
		Set   Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// TODO late init
	set, err := NewIntSetterFromConfig(cc.Param, cc.Set)
	if err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}

	o := &floatToIntProvider{
		set: set,
	}

	return o, nil
}

var _ SetFloatProvider = (*floatToIntProvider)(nil)

func (o *floatToIntProvider) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		return o.set(int64(val))
	}
}
