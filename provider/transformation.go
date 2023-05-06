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
		if cc.Type == "bool" {
			f, err := NewBoolGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					v, err := f()
					return v, err
				},
			})
		} else if cc.Type == "int" {
			f, err := NewIntGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					v, err := f()
					return v, err
				},
			})
		} else if cc.Type == "float" {
			f, err := NewFloatGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					v, err := f()
					return v, err
				},
			})
		} else {
			f, err := NewStringGetterFromConfig(cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			in = append(in, InTransformation{
				name: cc.Name,
				function: func() (any, error) {
					v, err := f()
					return v, err
				},
			})
		}
	}
	return in, nil
}

func ConvertOutFunctions(outConfig []TransformationConfig) ([]OutTransformation, error) {
	out := []OutTransformation{}
	for _, cc := range outConfig {
		name := cc.Name
		if cc.Type == "bool" {
			f, err := NewBoolSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					return f(v.(bool))
				},
			})
		} else if cc.Type == "int" {
			f, err := NewIntSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					return f(v.(int64))
				},
			})
		} else if cc.Type == "float" {
			f, err := NewFloatSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					return f(v.(float64))
				},
			})
		} else {
			f, err := NewStringSetterFromConfig(cc.Name, cc.Config)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			out = append(out, OutTransformation{
				name: cc.Name,
				Type: cc.Type,
				function: func(v any) error {
					return f(v.(string))
				},
			})
		}
	}
	return out, nil
}
