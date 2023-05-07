package provider

import (
	"fmt"
)

type (
	OutTransformationProvider[T any] interface {
		convertToInt(T) (int64, error)
		convertToString(T) (string, error)
		convertToFloat(T) (float64, error)
		convertToBool(T) (bool, error)
		outTransformations() []OutTransformation
	}
	InTransformationProvider interface {
		setParam(string, any) error
		inTransformations() []InTransformation
	}
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
	name, typ string
	function  func(any) error
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
				b, ok := v.(bool)
				if !ok {
					return fmt.Errorf("%s: could not convert %v to bool", cc.Name, b)
				}
				return ff(b)
			}

		case "int":
			ff, err := NewIntSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				b, ok := v.(int64)
				if !ok {
					return fmt.Errorf("%s: could not convert %v to int", cc.Name, b)
				}
				return ff(b)
			}

		case "float":
			ff, err := NewFloatSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				b, ok := v.(float64)
				if !ok {
					return fmt.Errorf("%s: could not convert %v to float", cc.Name, b)
				}
				return ff(b)
			}

		case "string":
			ff, err := NewStringSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", cc.Name, err)
			}

			f = func(v any) error {
				b, ok := v.(string)
				if !ok {
					return fmt.Errorf("%s: could not convert %v to string", cc.Name, b)
				}
				return ff(b)
			}

		default:
			return nil, fmt.Errorf("%s: invalid type %s", cc.Name, cc.Type)
		}

		out = append(out, OutTransformation{
			name:     cc.Name,
			typ:      cc.Type,
			function: f,
		})
	}

	return out, nil
}

func handleInTransformation[P InTransformationProvider](p P) error {
	for _, cc := range p.inTransformations() {
		val, err := cc.function()

		if err == nil {
			err = p.setParam(cc.name, val)
		}

		if err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}
	}

	return nil
}

func handleOutTransformation[P OutTransformationProvider[A], A any](p P, v A) error {
	for _, cc := range p.outTransformations() {
		var (
			vv  any
			err error
		)

		switch cc.typ {
		case "bool":
			vv, err = p.convertToBool(v)
		case "int":
			vv, err = p.convertToInt(v)
		case "float":
			vv, err = p.convertToFloat(v)
		case "string":
			vv, err = p.convertToString(v)
		default:
			return fmt.Errorf("%s: invalid type %s", cc.name, cc.typ)
		}

		if err == nil {
			err = cc.function(vv)
		}

		if err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}
	}

	return nil
}
