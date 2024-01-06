package meter

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
)

type meterRegistry map[string]func(map[string]interface{}) (api.Meter, error)

func (r meterRegistry) Add(name string, factory func(map[string]interface{}) (api.Meter, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate meter type: %s", name))
	}
	r[name] = factory
}

func (r meterRegistry) Get(name string) (func(map[string]interface{}) (api.Meter, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("invalid meter type: %s", name)
	}
	return factory, nil
}

var registry meterRegistry = make(map[string]func(map[string]interface{}) (api.Meter, error))

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (api.Meter, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(other)
	if err != nil {
		err = fmt.Errorf("cannot create meter type '%s': %w", typ, err)
	}

	return v, err
}
