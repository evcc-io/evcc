package provider

import (
	"context"
	"fmt"
	"strings"
)

type transformationConfig struct {
	Name, Type string
	Config     Config
}

type inputTransformation struct {
	name     string
	function func() (any, error)
}

type outputTransformation struct {
	name     string
	function func(any) error
}

func configureInputs(ctx context.Context, inConfig []transformationConfig) ([]inputTransformation, error) {
	var in []inputTransformation

	for _, cc := range inConfig {
		var f func() (any, error)

		switch strings.ToLower(cc.Type) {
		case "bool":
			ff, err := NewBoolGetterFromConfig(ctx, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		case "int":
			ff, err := NewIntGetterFromConfig(ctx, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		case "float":
			ff, err := NewFloatGetterFromConfig(ctx, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		case "string":
			ff, err := NewStringGetterFromConfig(ctx, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		default:
			return nil, fmt.Errorf("%s: invalid type %s", cc.Name, cc.Type)
		}

		in = append(in, inputTransformation{
			name:     cc.Name,
			function: f,
		})
	}
	return in, nil
}

func configureOutputs(ctx context.Context, outConfig []transformationConfig) ([]outputTransformation, error) {
	var out []outputTransformation

	for _, cc := range outConfig {
		var f func(v any) error

		switch strings.ToLower(cc.Type) {
		case "bool":
			ff, err := NewBoolSetterFromConfig(ctx, cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(bool))
			}

		case "int":
			ff, err := NewIntSetterFromConfig(ctx, cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(int64))
			}

		case "float":
			ff, err := NewFloatSetterFromConfig(ctx, cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(float64))
			}

		case "string":
			ff, err := NewStringSetterFromConfig(ctx, cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(string))
			}

		default:
			return nil, fmt.Errorf("%s: invalid type %s", cc.Name, cc.Type)
		}

		out = append(out, outputTransformation{
			name:     cc.Name,
			function: f,
		})
	}

	return out, nil
}

func transformInputs(in []inputTransformation, set func(string, any) error) error {
	for _, cc := range in {
		val, err := cc.function()

		if err == nil {
			err = set(cc.name, val)
		}

		if err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}
	}

	return nil
}

func transformOutputs(out []outputTransformation, v any) error {
	for _, cc := range out {
		if v == nil {
			return fmt.Errorf("no value to transform")
		}

		if err := cc.function(v); err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}
	}

	return nil
}

// normalizeValue transforms compatible plugin return types to ensure only supported ones are used
func normalizeValue(val any) (any, error) {
	switch v := val.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case float32:
		return float64(v), nil
	case int64, float64, bool, string:
		return v, nil
	default:
		return nil, fmt.Errorf("type not supported: %T", val)
	}
}
