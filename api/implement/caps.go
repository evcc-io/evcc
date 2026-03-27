package implement

import (
	"reflect"

	"github.com/evcc-io/evcc/api"
)

func Implements[T any](c Capabilities, impl T) {
	c.add(reflect.TypeFor[T](), impl)
}

type Capabilities interface {
	api.Capable
	add(typ reflect.Type, impl any)
}

// Caps creates a capabilities store exposing the api.Capable interface
func Caps() Capabilities {
	return make(caps)
}

type caps map[reflect.Type]any

// Capability implements the api.Capable interface
func (caps caps) Capability(typ reflect.Type) (any, bool) {
	c, ok := caps[typ]
	return c, ok
}

func (caps caps) add(typ reflect.Type, impl any) {
	caps[typ] = impl
}
