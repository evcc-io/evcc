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
	name, Type string
	function   func(any) error
}

func ConvertInFunctions(inConfig []TransformationConfig) ([]InTransformation, error) {
	in := []InTransformation{}
	for _, cc := range inConfig {
		name := cc.Name
		switch cc.Type {
		case "bool":
			f, err := NewBoolGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					return f()
				},
			})
		case "int":
			f, err := NewIntGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					return f()
				},
			})
		case "float":
			f, err := NewFloatGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					return f()
				},
			})
		case "string":
			f, err := NewStringGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					return f()
				},
			})
		default:
			return nil, fmt.Errorf("%s: Could not find converter for %s", name, cc.Type)
		}
	}
	return in, nil
}

func ConvertOutFunctions(outConfig []TransformationConfig) ([]OutTransformation, error) {
	out := []OutTransformation{}
	for _, cc := range outConfig {
		name := cc.Name
		switch cc.Type {
		case "bool":
			f, err := NewBoolSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					b, ok := v.(bool)
					if !ok {
						return fmt.Errorf("%s: Could not convert %v to bool", name, b)
					}
					return f(b)
				},
			})
		case "int":
			f, err := NewIntSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					b, ok := v.(int64)
					if !ok {
						return fmt.Errorf("%s: Could not convert %v to int", name, b)
					}
					return f(b)
				},
			})
		case "float":
			f, err := NewFloatSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					b, ok := v.(float64)
					if !ok {
						return fmt.Errorf("%s: Could not convert %v to float", name, b)
					}
					return f(b)
				},
			})
		case "string":
			f, err := NewStringSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					b, ok := v.(string)
					if !ok {
						return fmt.Errorf("%s: Could not convert %v to string", name, b)
					}
					return f(b)
				},
			})
		default:
			return nil, fmt.Errorf("%s: Could not find converter for %s", name, cc.Type)
		}
	}
	return out, nil
}
