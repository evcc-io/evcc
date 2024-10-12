package provider

import (
	"context"
	"strings"

	"github.com/evcc-io/evcc/util"
)

type ignoreProvider struct {
	ctx context.Context
	str string
	set Config
}

func init() {
	registry.AddCtx("ignore", NewIgnoreFromConfig)
}

// NewIgnoreFromConfig creates const provider
func NewIgnoreFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Error string
		Set   Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &ignoreProvider{
		ctx: ctx,
		str: cc.Error,
		set: cc.Set,
	}

	return o, nil
}

var _ SetIntProvider = (*ignoreProvider)(nil)

func ignoreError[T any](fun func(T) error, match string) func(T) error {
	return func(val T) error {
		err := fun(val)
		if err != nil && strings.HasPrefix(err.Error(), match) {
			err = nil
		}
		return err
	}
}

func (o *ignoreProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(o.ctx, param, o.set)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.str), nil
}

var _ SetFloatProvider = (*ignoreProvider)(nil)

func (o *ignoreProvider) FloatSetter(param string) (func(float64) error, error) {
	set, err := NewFloatSetterFromConfig(o.ctx, param, o.set)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.str), nil
}

var _ SetBoolProvider = (*ignoreProvider)(nil)

func (o *ignoreProvider) BoolSetter(param string) (func(bool) error, error) {
	set, err := NewBoolSetterFromConfig(o.ctx, param, o.set)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.str), nil
}

var _ SetBytesProvider = (*ignoreProvider)(nil)

func (o *ignoreProvider) BytesSetter(param string) (func([]byte) error, error) {
	set, err := NewBytesSetterFromConfig(o.ctx, param, o.set)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.str), nil
}
