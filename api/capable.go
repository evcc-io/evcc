package api

import "reflect"

// Capable is implemented by decorated types that support dynamic capability lookup.
// This allows O(n) decorator code instead of O(2^n) combinatorial struct generation.
type Capable interface {
	Capability(typ reflect.Type) (any, bool)
}

// Cap checks whether v implements interface T, either directly (via Go type assertion)
// or through the Capable registry (for decorated types) and returns the interface same
// as a Go type assertion would do.
func Cap[T any](v any) (T, bool) {
	// fast path: direct type assertion (works for non-decorated concrete types)
	if t, ok := v.(T); ok {
		return t, ok
	}

	// slow path: capability registry lookup (for decorated types)
	if c, ok := v.(Capable); ok {
		if impl, ok := c.Capability(reflect.TypeFor[T]()); ok {
			return impl.(T), true
		}
	}

	var zero T
	return zero, false
}

// HasCap checks whether v implements interface T, either directly (via Go type assertion)
// or through the Capable registry (for decorated types).
func HasCap[T any](v any) bool {
	_, ok := Cap[T](v)
	return ok
}
