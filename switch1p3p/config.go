package switch1p3p

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
)

type switch1p3pRegistry map[string]func(map[string]interface{}) (api.ChargePhases, error)

func (r switch1p3pRegistry) Add(name string, factory func(map[string]interface{}) (api.ChargePhases, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate switch1p3p type: %s", name))
	}
	r[name] = factory
}

func (r switch1p3pRegistry) Get(name string) (func(map[string]interface{}) (api.ChargePhases, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("switch1p3p type not registered: %s", name)
	}
	return factory, nil
}

var registry switch1p3pRegistry = make(map[string]func(map[string]interface{}) (api.ChargePhases, error))

// NewFromConfig creates switch from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.ChargePhases, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create switch1p3p '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid switch1p3p type: %s", typ)
	}

	return
}
