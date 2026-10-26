package config

import (
	"context"
	"errors"
	"fmt"
)

// Ref references a configured device by name or holds an inline typed config (yaml style nesting)
type Ref struct {
	Name  string `mapstructure:"-"`
	Type  string
	Other map[string]any `mapstructure:",remain"`
}

// UnmarshalText decodes a plain device name
func (r *Ref) UnmarshalText(text []byte) error {
	r.Name = string(text)
	return nil
}

// Resolve returns the referenced device instance, instantiating inline configs
func Resolve[T any](ctx context.Context, h Handler[T], ref Ref, newFromConf func(context.Context, string, map[string]any) (T, error)) (T, error) {
	var zero T

	if ref.Name != "" {
		dev, err := h.ByName(ref.Name)
		if err != nil {
			return zero, err
		}
		if instance := dev.Instance(); any(instance) != nil {
			return instance, nil
		}
		return zero, fmt.Errorf("%s not available (disabled or failed)", ref.Name)
	}

	if ref.Type == "" {
		return zero, errors.New("missing device reference")
	}

	return newFromConf(ctx, ref.Type, ref.Other)
}
