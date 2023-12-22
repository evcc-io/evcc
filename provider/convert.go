package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type convertProvider struct {
	Convert string
	Set     Config
}

func init() {
	registry.Add("convert", NewConvertFromConfig)
}

// NewConvertFromConfig creates type conversion provider
func NewConvertFromConfig(other map[string]interface{}) (Provider, error) {
	var cc convertProvider

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}

var _ SetFloatProvider = (*convertProvider)(nil)

func (o *convertProvider) FloatSetter(param string) (func(float64) error, error) {
	if o.Convert != "float2int" {
		return nil, fmt.Errorf("convert: invalid conversion: %s", o.Convert)
	}

	set, err := NewIntSetterFromConfig(param, o.Set)

	return func(val float64) error {
		return set(int64(val))
	}, err
}

var _ SetIntProvider = (*convertProvider)(nil)

func (o *convertProvider) IntSetter(param string) (func(int64) error, error) {
	if o.Convert != "int2float" {
		return nil, fmt.Errorf("convert: invalid conversion: %s", o.Convert)
	}

	set, err := NewFloatSetterFromConfig(param, o.Set)

	return func(val int64) error {
		return set(float64(val))
	}, err
}
