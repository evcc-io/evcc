package implement

import (
	"reflect"

	"github.com/evcc-io/evcc/api"
)

// Has registers impl as a capability on c. It panics if impl is nil.
func Has[T any](c Caps, impl T) {
	typ := reflect.TypeFor[T]()
	if isNil(impl) {
		panic("implement: nil " + typ.String())
	}
	c.add(typ, impl)
}

// May registers impl as a capability on c. If impl is nil, it is silently ignored.
func May[T any](c Caps, impl T) {
	if isNil(impl) {
		return
	}
	c.add(reflect.TypeFor[T](), impl)
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

type Caps interface {
	api.Capable
	add(typ reflect.Type, impl any)
}

// New creates a capabilities store exposing the api.Capable interface
func New() Caps {
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
