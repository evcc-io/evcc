package provider

import (
	"fmt"
)

type TransformationConfig struct {
	Name, Type string
	Config     Config
}

type InTransformation struct {
	name     string
	function func() (any, error)
}

type OutTransformation struct {
	name     string
	function func(any) error
}

func ConvertInFunctions(inConfig []TransformationConfig) ([]InTransformation, error) {
	var in []InTransformation

	for _, cc := range inConfig {
		var f func() (any, error)

		switch cc.Type {
		case "bool":
			ff, err := NewBoolGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		case "int":
			ff, err := NewIntGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		case "float":
			ff, err := NewFloatGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		case "string":
			ff, err := NewStringGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}
			f = func() (any, error) { return ff() }

		default:
			return nil, fmt.Errorf("%s: Could not find converter for %s", cc.Name, cc.Type)
		}

		in = append(in, InTransformation{
			name:     cc.Name,
			function: f,
		})
	}
	return in, nil
}

func ConvertOutFunctions(outConfig []TransformationConfig) ([]OutTransformation, error) {
	var out []OutTransformation

	for _, cc := range outConfig {
		var f func(v any) error

		switch cc.Type {
		case "bool":
			ff, err := NewBoolSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(bool))
			}

		case "int":
			ff, err := NewIntSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(int64))
			}

		case "float":
			ff, err := NewFloatSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(float64))
			}

		case "string":
			ff, err := NewStringSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				return ff(v.(string))
			}

		default:
			return nil, fmt.Errorf("%s: invalid type %s", cc.Name, cc.Type)
		}

		out = append(out, OutTransformation{
			name:     cc.Name,
			function: f,
		})
	}

	return out, nil
}

func handleInTransformation(in []InTransformation, set func(string, any) error) error {
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

func handleOutTransformation(out []OutTransformation, v any) error {
	for _, cc := range out {
		if err := cc.function(v); err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}
	}

	return nil
}
