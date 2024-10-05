package provider

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type convertProvider struct {
	ctx     context.Context
	Convert string
	Set     Config
}

func init() {
	registry.AddCtx("convert", NewConvertFromConfig)
}

// NewConvertFromConfig creates type conversion provider
func NewConvertFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	cc := convertProvider{
		ctx: ctx,
	}

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

	set, err := NewIntSetterFromConfig(o.ctx, param, o.Set)

	return func(val float64) error {
		return set(int64(val))
	}, err
}

var _ SetIntProvider = (*convertProvider)(nil)

func (o *convertProvider) IntSetter(param string) (func(int64) error, error) {
	if o.Convert != "int2float" {
		return nil, fmt.Errorf("convert: invalid conversion: %s", o.Convert)
	}

	set, err := NewFloatSetterFromConfig(o.ctx, param, o.Set)

	return func(val int64) error {
		return set(float64(val))
	}, err
}
