package provider

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/server/config"
)

type typeDesc struct {
	factory func(map[string]interface{}) (IntProvider, error)
	config  config.Type
}

type typeRegistry map[string]typeDesc

var registry = make(typeRegistry)

// Types exports the public configuration types
func Types() (types []config.Type) {
	for _, typ := range registry {
		if typ.config.Config != nil {
			types = append(types, typ.config)
		}
	}

	return types
}

// Add adds a meter description to the registry
func (r typeRegistry) Add(name, label string, factory func(map[string]interface{}) (IntProvider, error), defaults interface{}) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate meter type: %s", name))
	}

	desc := typeDesc{
		factory: factory,
		config: config.Type{
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				return factory(cfg)
			},
			Type:   name,
			Label:  label,
			Config: defaults,
		},
	}

	r[name] = desc
}

// Get retrieves a meter description from the registry
func (r typeRegistry) Get(name string) (typeDesc, error) {
	desc, exists := r[name]
	if !exists {
		return typeDesc{}, fmt.Errorf("invalid plugin type: %s", name)
	}
	return desc, nil
}

// Config is the general provider config
type Config struct {
	Type  string
	Other map[string]interface{} `mapstructure:",remain"`
}

// NewIntGetterFromConfig creates a IntGetter from config
func NewIntGetterFromConfig(config Config) (res func() (int64, error), err error) {
	desc, err := registry.Get(strings.ToLower(config.Type))
	if err == nil {
		var provider IntProvider
		provider, err = desc.factory(config.Other)

		if err == nil {
			res = provider.IntGetter()
		}
	}

	if err == nil && res == nil {
		err = fmt.Errorf("invalid plugin type: %s", config.Type)
	}

	return
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(config Config) (res func() (float64, error), err error) {
	desc, err := registry.Get(strings.ToLower(config.Type))
	if err == nil {
		var provider IntProvider
		provider, err = desc.factory(config.Other)

		if prov, ok := provider.(FloatProvider); ok {
			res = prov.FloatGetter()
		}
	}

	if err == nil && res == nil {
		err = fmt.Errorf("invalid plugin type: %s", config.Type)
	}

	return
}

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(config Config) (res func() (string, error), err error) {
	desc, err := registry.Get(strings.ToLower(config.Type))
	if err == nil {
		var provider IntProvider
		provider, err = desc.factory(config.Other)

		if prov, ok := provider.(StringProvider); ok {
			res = prov.StringGetter()
		}
	}

	if err == nil && res == nil {
		err = fmt.Errorf("invalid plugin type: %s", config.Type)
	}

	return
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(config Config) (res func() (bool, error), err error) {
	desc, err := registry.Get(strings.ToLower(config.Type))
	if err == nil {
		var provider IntProvider
		provider, err = desc.factory(config.Other)

		if prov, ok := provider.(BoolProvider); ok {
			res = prov.BoolGetter()
		}
	}

	if err == nil && res == nil {
		err = fmt.Errorf("invalid plugin type: %s", config.Type)
	}

	return
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(param string, config Config) (res func(int64) error, err error) {
	desc, err := registry.Get(strings.ToLower(config.Type))
	if err == nil {
		var provider IntProvider
		provider, err = desc.factory(config.Other)

		if prov, ok := provider.(SetIntProvider); ok {
			res = prov.IntSetter(param)
		}
	}

	if err == nil && res == nil {
		err = fmt.Errorf("invalid plugin type: %s", config.Type)
	}

	return
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(param string, config Config) (res func(bool) error, err error) {
	desc, err := registry.Get(strings.ToLower(config.Type))
	if err == nil {
		var provider IntProvider
		provider, err = desc.factory(config.Other)

		if prov, ok := provider.(SetBoolProvider); ok {
			res = prov.BoolSetter(param)
		}
	}

	if err == nil && res == nil {
		err = fmt.Errorf("invalid plugin type: %s", config.Type)
	}

	return
}
