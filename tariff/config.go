package tariff

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
)

type tariffRegistry map[string]func(map[string]interface{}) (api.Tariff, error)

func (r tariffRegistry) Add(name string, factory func(map[string]interface{}) (api.Tariff, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate tariff type: %s", name))
	}
	r[name] = factory
}

func (r tariffRegistry) Get(name string) (func(map[string]interface{}) (api.Tariff, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("tariff type not registered: %s", name)
	}
	return factory, nil
}

var registry tariffRegistry = make(map[string]func(map[string]interface{}) (api.Tariff, error))

// NewFromConfig creates tariff from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Tariff, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create tariff '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid tariff type: %s", typ)
	}

	return
}
