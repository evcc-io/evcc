package plugin

import (
	"context"
)

func init() {
	registry.AddCtx("noop", NewNoopFromConfig)
}

type noopPlugin struct{}

func NewNoopFromConfig(_ context.Context, _ map[string]any) (Plugin, error) {
	return &noopPlugin{}, nil
}

func noop[T any]() func(T) error {
	return func(T) error { return nil }
}

func (o *noopPlugin) IntSetter(_ string) (func(int64) error, error) { return noop[int64](), nil }
func (o *noopPlugin) FloatSetter(_ string) (func(float64) error, error) { return noop[float64](), nil }
func (o *noopPlugin) BoolSetter(_ string) (func(bool) error, error) { return noop[bool](), nil }
