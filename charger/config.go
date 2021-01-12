package charger

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/server/config"
)

type typeDesc struct {
	factory func(map[string]interface{}) (api.Charger, error)
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
func (r typeRegistry) Add(name, label string, factory func(map[string]interface{}) (api.Charger, error), defaults interface{}) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate charger type: %s", name))
	}

	typ := typeDesc{
		factory: factory,
		config: config.Type{
			Type:   name,
			Label:  label,
			Config: defaults,
		},
	}

	r[name] = typ
}

func (r typeRegistry) Get(name string) (func(map[string]interface{}) (api.Charger, error), error) {
	typ, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("charger type not registered: %s", name)
	}
	return typ.factory, nil
}

// NewFromConfig creates charger from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Charger, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create type '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid charger type: %s", typ)
	}

	return
}
