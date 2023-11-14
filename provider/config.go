package provider

import (
	"fmt"
)

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

type providerRegistry map[string]func(map[string]interface{}) (Provider, error)

func (r providerRegistry) Add(name string, factory func(map[string]interface{}) (Provider, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate plugin type: %s", name))
	}
	r[name] = factory
}

func (r providerRegistry) Get(name string) (func(map[string]interface{}) (Provider, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("invalid plugin source: %s", name)
	}
	return factory, nil
}

var registry providerRegistry = make(map[string]func(map[string]interface{}) (Provider, error))

// Config is the general provider config
type Config struct {
	Source string
	Other  map[string]interface{} `mapstructure:",remain"`
}

// NewIntGetterFromConfig creates a IntGetter from config
func NewIntGetterFromConfig(config Config) (func() (int64, error), error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(IntProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type int: %s", config.Source)
	}

	return prov.IntGetter()
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(config Config) (func() (float64, error), error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(FloatProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type float: %s", config.Source)
	}

	return prov.FloatGetter()
}

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(config Config) (func() (string, error), error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(StringProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type string: %s", config.Source)
	}

	return prov.StringGetter()
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(config Config) (func() (bool, error), error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(BoolProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type bool: %s", config.Source)
	}

	return prov.BoolGetter()
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(param string, config Config) (func(int64) error, error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(SetIntProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type int: %s", config.Source)
	}

	return prov.IntSetter(param)
}

// NewFloatSetterFromConfig creates a FloatSetter from config
func NewFloatSetterFromConfig(param string, config Config) (func(float642 float64) error, error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(SetFloatProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type float: %s", config.Source)
	}

	return prov.FloatSetter(param)
}

// NewStringSetterFromConfig creates a StringSetter from config
func NewStringSetterFromConfig(param string, config Config) (func(string) error, error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(SetStringProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type string: %s", config.Source)
	}

	return prov.StringSetter(param)
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(param string, config Config) (func(bool) error, error) {
	factory, err := registry.Get(config.Source)
	if err != nil {
		return nil, err
	}

	provider, err := factory(config.Other)
	if err != nil {
		return nil, err
	}

	prov, ok := provider.(SetBoolProvider)
	if !ok {
		return nil, fmt.Errorf("invalid plugin source for type bool: %s", config.Source)
	}

	return prov.BoolSetter(param)
}
