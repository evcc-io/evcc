package charger

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/server/config"
)

var configTypes []config.Type

func registerConfig(typ, name string, defaults interface{}, rank ...int) {
	typeConfig := config.Type{
		Type:   typ,
		Name:   name,
		Config: defaults,
	}

	if len(rank) > 0 {
		typeConfig.Rank = rank[0]
	}

	configTypes = append(configTypes, typeConfig)
}

func ConfigTypes() []config.Type {
	return configTypes
}

type chargerRegistry map[string]func(map[string]interface{}) (api.Charger, error)

func (r chargerRegistry) Add(name string, factory func(map[string]interface{}) (api.Charger, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate charger type: %s", name))
	}
	r[name] = factory
}

func (r chargerRegistry) Get(name string) (func(map[string]interface{}) (api.Charger, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("charger type not registered: %s", name)
	}
	return factory, nil
}

var registry chargerRegistry = make(map[string]func(map[string]interface{}) (api.Charger, error))

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
