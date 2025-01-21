package provider

import (
	"context"
	"fmt"

	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[Provider]("plugin")

// provider types
type (
	Provider interface{}
	Getters  interface {
		StringProvider
		FloatProvider
		IntProvider
		BoolProvider
	}
	StringProvider interface {
		StringGetter() (func() (string, error), error)
	}
	FloatProvider interface {
		FloatGetter() (func() (float64, error), error)
	}
	IntProvider interface {
		IntGetter() (func() (int64, error), error)
	}
	BoolProvider interface {
		BoolGetter() (func() (bool, error), error)
	}
	SetStringProvider interface {
		StringSetter(param string) (func(string) error, error)
	}
	SetFloatProvider interface {
		FloatSetter(param string) (func(float64) error, error)
	}
	SetIntProvider interface {
		IntSetter(param string) (func(int64) error, error)
	}
	SetBoolProvider interface {
		BoolSetter(param string) (func(bool) error, error)
	}
	SetBytesProvider interface {
		BytesSetter(param string) (func([]byte) error, error)
	}
)

// Config is the general provider config
type Config struct {
	Source string
	Other  map[string]any `mapstructure:",remain" yaml:",inline"`
}

func provider[T any](typ string, ctx context.Context, config *Config) (T, error) {
	var zero T

	if config == nil {
		return zero, nil
	}

	factory, err := registry.Get(config.Source)
	if err != nil {
		return zero, err
	}

	provider, err := factory(ctx, config.Other)
	if err != nil {
		return zero, err
	}

	prov, ok := provider.(T)
	if !ok {
		return zero, fmt.Errorf("invalid plugin source for type %s: %s", typ, config.Source)
	}

	return prov, nil
}

func (c *Config) IntGetter(ctx context.Context) (func() (int64, error), error) {
	prov, err := provider[IntProvider]("int", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.IntGetter()
}

func (c *Config) FloatGetter(ctx context.Context) (func() (float64, error), error) {
	prov, err := provider[FloatProvider]("float", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.FloatGetter()
}

func (c *Config) StringGetter(ctx context.Context) (func() (string, error), error) {
	prov, err := provider[StringProvider]("string", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.StringGetter()
}

func (c *Config) BoolGetter(ctx context.Context) (func() (bool, error), error) {
	prov, err := provider[BoolProvider]("bool", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.BoolGetter()
}

func (c *Config) IntSetter(ctx context.Context, param string) (func(int64) error, error) {
	prov, err := provider[SetIntProvider]("int", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.IntSetter(param)
}

func (c *Config) FloatSetter(ctx context.Context, param string) (func(float642 float64) error, error) {
	prov, err := provider[SetFloatProvider]("float", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.FloatSetter(param)
}

func (c *Config) StringSetter(ctx context.Context, param string) (func(string) error, error) {
	prov, err := provider[SetStringProvider]("string", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.StringSetter(param)
}

func (c *Config) BoolSetter(ctx context.Context, param string) (func(bool) error, error) {
	prov, err := provider[SetBoolProvider]("bool", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.BoolSetter(param)
}

func (c *Config) BytesSetter(ctx context.Context, param string) (func([]byte) error, error) {
	prov, err := provider[SetBytesProvider]("bytes", ctx, c)
	if prov == nil || err != nil {
		return nil, err
	}

	return prov.BytesSetter(param)
}
