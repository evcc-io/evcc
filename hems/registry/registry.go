package registry

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/evcc-io/evcc/core/site"
)

type (
	// TODO check is this can be abstracted into a generic type parameter
	factoryFunc[T any] func(context.Context, map[string]any, site.API) (T, error)

	registry[T any] struct {
		typ  string
		data map[string]factoryFunc[T]
	}
)

func (r registry[T]) Add(name string, factory func(map[string]any, site.API) (T, error)) {
	r.AddCtx(name, func(_ context.Context, cc map[string]any, site site.API) (T, error) {
		return factory(cc, site)
	})
}

func (r registry[T]) AddCtx(name string, factory factoryFunc[T]) {
	if _, exists := r.data[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate %s type: %s", r.typ, name))
	}
	r.data[name] = factory
}

func (r registry[T]) Get(name string) (factoryFunc[T], error) {
	factory, exists := r.data[name]
	if !exists {
		return nil, fmt.Errorf("invalid %s type: %s", r.typ, name)
	}
	return factory, nil
}

func (r registry[T]) Types() []string {
	return slices.Sorted(maps.Keys(r.data))
}

func New[T any](typ string) registry[T] {
	return registry[T]{
		typ:  typ,
		data: make(map[string]factoryFunc[T]),
	}
}
