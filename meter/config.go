package meter

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/server/config"
)

type typeDesc struct {
	factory func(map[string]interface{}) (api.Meter, error)
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

func GetConfig(name string) (config.Type, error) {
	desc, err := registry.Get(name)
	if err == nil {
		return desc.config, err
	}

	return config.Type{}, err
}

func (r typeRegistry) Add(name, label string, factory func(map[string]interface{}) (api.Meter, error), defaults interface{}) {
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

func (r typeRegistry) Get(name string) (typeDesc, error) {
	desc, exists := r[name]
	if !exists {
		return typeDesc{}, fmt.Errorf("meter type not registered: %s", name)
	}
	return desc, nil
}

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Meter, err error) {
	desc, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = desc.factory(other); err != nil {
			err = fmt.Errorf("cannot create type '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid meter type: %s", typ)
	}

	return
}
