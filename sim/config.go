package simulation

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
)

type simRegistry map[string]func(*Actors, map[string]interface{}) (api.Updateable, error)

func (r simRegistry) Add(name string, factory func(*Actors, map[string]interface{}) (api.Updateable, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate simulator type: %s", name))
	}
	r[name] = factory
}

func (r simRegistry) Get(name string) (func(*Actors, map[string]interface{}) (api.Updateable, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("simulator type not registered: %s", name)
	}
	return factory, nil
}

var registry simRegistry = make(map[string]func(*Actors, map[string]interface{}) (api.Updateable, error))

// NewFromConfig creates simulator from configuration
func NewFromConfig(actors *Actors, typ string, other map[string]interface{}) (v api.Updateable, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(actors, other); err != nil {
			err = fmt.Errorf("cannot create simulator '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid simulator type: %s", typ)
	}

	return
}
