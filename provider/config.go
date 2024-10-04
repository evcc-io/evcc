package provider

import (
	"context"
	"fmt"

	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[Provider]("plugin")

// provider types
type (
	Provider    interface{}
	IntProvider interface {
		IntGetter() (func() (int64, error), error)
	}
	StringProvider interface {
		StringGetter() (func() (string, error), error)
	}
	FloatProvider interface {
		FloatGetter() (func() (float64, error), error)
	}
	BoolProvider interface {
		BoolGetter() (func() (bool, error), error)
	}
	SetIntProvider interface {
		IntSetter(param string) (func(int64) error, error)
	}
	SetStringProvider interface {
		StringSetter(param string) (func(string) error, error)
	}
	SetFloatProvider interface {
		FloatSetter(param string) (func(float64) error, error)
	}
	SetBoolProvider interface {
		BoolSetter(param string) (func(bool) error, error)
	}
)

// Config is the general provider config
type Config struct {
	Source string
	Other  map[string]any `mapstructure:",remain" yaml:",inline"`
}

func provider[T any](typ string, config Config) (T, error) {
	var zero T

	factory, err := registry.Get(config.Source)
	if err != nil {
		return zero, err
	}

	provider, err := factory(context.TODO(), config.Other)
	if err != nil {
		return zero, err
	}

	prov, ok := provider.(T)
	if !ok {
		return zero, fmt.Errorf("invalid plugin source for type %s: %s", typ, config.Source)
	}

	return prov, nil
}

// NewIntGetterFromConfig creates a IntGetter from config
func NewIntGetterFromConfig(config Config) (func() (int64, error), error) {
	prov, err := provider[IntProvider]("int", config)
	if err != nil {
		return nil, err
	}

	return prov.IntGetter()
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(config Config) (func() (float64, error), error) {
	prov, err := provider[FloatProvider]("float", config)
	if err != nil {
		return nil, err
	}

	return prov.FloatGetter()
}

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(config Config) (func() (string, error), error) {
	prov, err := provider[StringProvider]("string", config)
	if err != nil {
		return nil, err
	}

	return prov.StringGetter()
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(config Config) (func() (bool, error), error) {
	prov, err := provider[BoolProvider]("bool", config)
	if err != nil {
		return nil, err
	}

	return prov.BoolGetter()
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(param string, config Config) (func(int64) error, error) {
	prov, err := provider[SetIntProvider]("int", config)
	if err != nil {
		return nil, err
	}

	return prov.IntSetter(param)
}

// NewFloatSetterFromConfig creates a FloatSetter from config
func NewFloatSetterFromConfig(param string, config Config) (func(float642 float64) error, error) {
	prov, err := provider[SetFloatProvider]("float", config)
	if err != nil {
		return nil, err
	}

	return prov.FloatSetter(param)
}

// NewStringSetterFromConfig creates a StringSetter from config
func NewStringSetterFromConfig(param string, config Config) (func(string) error, error) {
	prov, err := provider[SetStringProvider]("string", config)
	if err != nil {
		return nil, err
	}

	return prov.StringSetter(param)
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(param string, config Config) (func(bool) error, error) {
	prov, err := provider[SetBoolProvider]("bool", config)
	if err != nil {
		return nil, err
	}

	return prov.BoolSetter(param)
}
