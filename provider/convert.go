package provider

import (
	"encoding/binary"
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
	switch o.Convert {
	case "float2int":
		set, err := NewIntSetterFromConfig(param, o.Set)

		return func(val float64) error {
			return set(int64(val))
		}, err

	default:
		return nil, fmt.Errorf("convert: invalid conversion: %s", o.Convert)
	}
}

var _ SetIntProvider = (*convertProvider)(nil)

func (o *convertProvider) IntSetter(param string) (func(int64) error, error) {
	switch o.Convert {
	case "int2float":
		set, err := NewFloatSetterFromConfig(param, o.Set)

		return func(val int64) error {
			return set(float64(val))
		}, err

	case "int2bytes":
		set, err := NewBytesSetterFromConfig(param, o.Set)

		return func(val int64) error {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(val))
			return set(b)
		}, err

	default:
		return nil, fmt.Errorf("convert: invalid conversion: %s", o.Convert)
	}
}
