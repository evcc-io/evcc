package meter

import (
	"fmt"
	"strings"

	"github.com/mark-sch/evcc/api"
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
		return nil, fmt.Errorf("meter type not registered: %s", name)
	}
	return factory, nil
}

var registry meterRegistry = make(map[string]func(map[string]interface{}) (api.Meter, error))

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Meter, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create type '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid meter type: %s", typ)
	}

	return
}
