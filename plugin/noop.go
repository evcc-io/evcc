package plugin

import (
	"context"
)

type noopPlugin struct{}

func init() {
	registry.AddCtx("noop", NewNoopFromConfig)
}

// NewNoopFromConfig creates noop provider
func NewNoopFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	return &noopPlugin{}, nil
}

// noop is the generic no-op setter function for noopPlugin
func noop[T comparable]() func(T) error {
	return func(val T) error {
		return nil
	}
}

var _ IntSetter = (*noopPlugin)(nil)

func (o *noopPlugin) IntSetter(param string) (func(int64) error, error) {
	return noop[int64](), nil
}

var _ FloatSetter = (*noopPlugin)(nil)

func (o *noopPlugin) FloatSetter(param string) (func(float64) error, error) {
	return noop[float64](), nil
}

var _ BoolSetter = (*noopPlugin)(nil)

func (o *noopPlugin) BoolSetter(param string) (func(bool) error, error) {
	return noop[bool](), nil
}
