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

func (c *Config) IntGetter(ctx context.Context) (func() (int64, error), error) {
	if c == nil {
		return nil, nil
	}
	return newIntGetterFromConfig(ctx, *c)
}

func (c *Config) FloatGetter(ctx context.Context) (func() (float64, error), error) {
	if c == nil {
		return nil, nil
	}
	return newFloatGetterFromConfig(ctx, *c)
}

func (c *Config) StringGetter(ctx context.Context) (func() (string, error), error) {
	if c == nil {
		return nil, nil
	}
	return newStringGetterFromConfig(ctx, *c)
}

func (c *Config) BoolGetter(ctx context.Context) (func() (bool, error), error) {
	if c == nil {
		return nil, nil
	}
	return newBoolGetterFromConfig(ctx, *c)
}

func (c *Config) IntSetter(ctx context.Context, param string) (func(int64) error, error) {
	if c == nil {
		return nil, nil
	}
	return newIntSetterFromConfig(ctx, param, *c)
}

func (c *Config) FloatSetter(ctx context.Context, param string) (func(float642 float64) error, error) {
	if c == nil {
		return nil, nil
	}
	return newFloatSetterFromConfig(ctx, param, *c)
}

func (c *Config) StringSetter(ctx context.Context, param string) (func(string) error, error) {
	if c == nil {
		return nil, nil
	}
	return newStringSetterFromConfig(ctx, param, *c)
}

func (c *Config) BoolSetter(ctx context.Context, param string) (func(bool) error, error) {
	if c == nil {
		return nil, nil
	}
	return newBoolSetterFromConfig(ctx, param, *c)
}

func (c *Config) BytesSetter(ctx context.Context, param string) (func([]byte) error, error) {
	if c == nil {
		return nil, nil
	}
	return newBytesSetterFromConfig(ctx, param, *c)
}

func provider[T any](typ string, ctx context.Context, config Config) (T, error) {
	var zero T

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

// newIntGetterFromConfig creates a IntGetter from config
func newIntGetterFromConfig(ctx context.Context, config Config) (func() (int64, error), error) {
	prov, err := provider[IntProvider]("int", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.IntGetter()
}

// newFloatGetterFromConfig creates a FloatGetter from config
func newFloatGetterFromConfig(ctx context.Context, config Config) (func() (float64, error), error) {
	prov, err := provider[FloatProvider]("float", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.FloatGetter()
}

// newStringGetterFromConfig creates a StringGetter from config
func newStringGetterFromConfig(ctx context.Context, config Config) (func() (string, error), error) {
	prov, err := provider[StringProvider]("string", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.StringGetter()
}

// newBoolGetterFromConfig creates a BoolGetter from config
func newBoolGetterFromConfig(ctx context.Context, config Config) (func() (bool, error), error) {
	prov, err := provider[BoolProvider]("bool", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.BoolGetter()
}

// newIntSetterFromConfig creates a IntSetter from config
func newIntSetterFromConfig(ctx context.Context, param string, config Config) (func(int64) error, error) {
	prov, err := provider[SetIntProvider]("int", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.IntSetter(param)
}

// newFloatSetterFromConfig creates a FloatSetter from config
func newFloatSetterFromConfig(ctx context.Context, param string, config Config) (func(float642 float64) error, error) {
	prov, err := provider[SetFloatProvider]("float", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.FloatSetter(param)
}

// newStringSetterFromConfig creates a StringSetter from config
func newStringSetterFromConfig(ctx context.Context, param string, config Config) (func(string) error, error) {
	prov, err := provider[SetStringProvider]("string", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.StringSetter(param)
}

// newBoolSetterFromConfig creates a BoolSetter from config
func newBoolSetterFromConfig(ctx context.Context, param string, config Config) (func(bool) error, error) {
	prov, err := provider[SetBoolProvider]("bool", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.BoolSetter(param)
}

// newBytesSetterFromConfig creates a BytesSetter from config
func newBytesSetterFromConfig(ctx context.Context, param string, config Config) (func([]byte) error, error) {
	prov, err := provider[SetBytesProvider]("bytes", ctx, config)
	if err != nil {
		return nil, err
	}

	return prov.BytesSetter(param)
}
