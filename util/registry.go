package util

import (
	"fmt"
)

type Registry[T any] map[string]func(map[string]interface{}) (T, error)

func (r Registry[T]) Add(name string, factory func(map[string]interface{}) (T, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate type: %s", name))
	}
	r[name] = factory
}

func (r Registry[T]) Get(name string) (func(map[string]interface{}) (T, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("type not registered: %s", name)
	}
	return factory, nil
}
