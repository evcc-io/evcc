package plugin

import (
	"context"
	"errors"
	"fmt"

	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[Plugin]("plugin")

// plugin types
type (
	Plugin  any
	Getters interface {
		StringGetter
		FloatGetter
		IntGetter
		BoolGetter
	}
	StringGetter interface {
		StringGetter() (func() (string, error), error)
	}
	FloatGetter interface {
		FloatGetter() (func() (float64, error), error)
	}
	IntGetter interface {
		IntGetter() (func() (int64, error), error)
	}
	BoolGetter interface {
		BoolGetter() (func() (bool, error), error)
	}
	StringSetter interface {
		StringSetter(param string) (func(string) error, error)
	}
	FloatSetter interface {
		FloatSetter(param string) (func(float64) error, error)
	}
	IntSetter interface {
		IntSetter(param string) (func(int64) error, error)
	}
	BoolSetter interface {
		BoolSetter(param string) (func(bool) error, error)
	}
	BytesSetter interface {
		BytesSetter(param string) (func([]byte) error, error)
	}
)

// Config is the general plugin config
type Config struct {
	Source string
	Other  map[string]any `mapstructure:",remain" yaml:",inline"`
}

func plugin[T any](typ string, ctx context.Context, config *Config) (T, error) {
	var zero T

	if config == nil {
		return zero, nil
	}

	if config.Source == "" {
		return zero, errors.New("missing plugin source")
	}

	factory, err := registry.Get(config.Source)
	if err != nil {
		return zero, err
	}

	plugin, err := factory(ctx, config.Other)
	if err != nil {
		return zero, err
	}

	prov, ok := plugin.(T)
	if !ok {
		return zero, fmt.Errorf("invalid plugin source for type %s: %s", typ, config.Source)
	}

	return prov, nil
}

func (c *Config) IntGetter(ctx context.Context) (func() (int64, error), error) {
	prov, err := plugin[IntGetter]("int", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.IntGetter()
}

func (c *Config) FloatGetter(ctx context.Context) (func() (float64, error), error) {
	prov, err := plugin[FloatGetter]("float", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.FloatGetter()
}

func (c *Config) StringGetter(ctx context.Context) (func() (string, error), error) {
	prov, err := plugin[StringGetter]("string", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.StringGetter()
}

func (c *Config) BoolGetter(ctx context.Context) (func() (bool, error), error) {
	prov, err := plugin[BoolGetter]("bool", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.BoolGetter()
}

func (c *Config) IntSetter(ctx context.Context, param string) (func(int64) error, error) {
	prov, err := plugin[IntSetter]("int", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.IntSetter(param)
}

func (c *Config) FloatSetter(ctx context.Context, param string) (func(float642 float64) error, error) {
	prov, err := plugin[FloatSetter]("float", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.FloatSetter(param)
}

func (c *Config) StringSetter(ctx context.Context, param string) (func(string) error, error) {
	prov, err := plugin[StringSetter]("string", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.StringSetter(param)
}

func (c *Config) BoolSetter(ctx context.Context, param string) (func(bool) error, error) {
	prov, err := plugin[BoolSetter]("bool", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.BoolSetter(param)
}

func (c *Config) BytesSetter(ctx context.Context, param string) (func([]byte) error, error) {
	prov, err := plugin[BytesSetter]("bytes", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.BytesSetter(param)
}
