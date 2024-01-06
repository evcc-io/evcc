package charger

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
)

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
		return nil, fmt.Errorf("invalid charger type: %s", name)
	}
	return factory, nil
}

var registry chargerRegistry = make(map[string]func(map[string]interface{}) (api.Charger, error))

// NewFromConfig creates charger from configuration
func NewFromConfig(typ string, other map[string]interface{}) (api.Charger, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(other)
	if err != nil {
		err = fmt.Errorf("cannot create charger type '%s': %w", typ, err)
	}

	return v, err
}
