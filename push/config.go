package push

import (
	"fmt"
	"strings"
)

// Messenger implements message sending
type Messenger interface {
	Send(title, msg string)
}

type senderRegistry map[string]func(map[string]interface{}) (Messenger, error)

func (r senderRegistry) Add(name string, factory func(map[string]interface{}) (Messenger, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate messenger type: %s", name))
	}
	r[name] = factory
}

func (r senderRegistry) Get(name string) (func(map[string]interface{}) (Messenger, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("messenger type not registered: %s", name)
	}
	return factory, nil
}

var registry senderRegistry = make(map[string]func(map[string]interface{}) (Messenger, error))

// NewFromConfig creates messenger from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v Messenger, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create messenger '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid messenger type: %s", typ)
	}

	return
}
