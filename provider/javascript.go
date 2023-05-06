package provider

import (
	"fmt"
	"github.com/evcc-io/evcc/provider/javascript"
	"github.com/evcc-io/evcc/util"
	"github.com/robertkrimen/otto"
)

// Javascript implements Javascript request provider
type Javascript struct {
	vm     *otto.Otto
	script string
	in     []TransformationConfig
	out    []TransformationConfig
}

type TransformationConfig struct {
	Name, Type string
	Config     Config
}

func init() {
	registry.Add("js", NewJavascriptProviderFromConfig)
}

// NewJavascriptProviderFromConfig creates a Javascript provider
func NewJavascriptProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		VM     string
		Script string
		In     []TransformationConfig
		Out    []TransformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := javascript.RegisteredVM(cc.VM, "")
	if err != nil {
		return nil, err
	}

	p := &Javascript{
		vm:     vm,
		script: cc.Script,
		in:     cc.In,
		out:    cc.Out,
	}

	return p, nil
}

// FloatGetter parses float from request
func (p *Javascript) FloatGetter() func() (float64, error) {
	return func() (res float64, err error) {
		if p.in != nil {
			err = transformGetter(p)
		}
		if err == nil {
			var v otto.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				res, err = v.ToFloat()
			}
		}
		return res, err
	}
}

// IntGetter parses int64 from request
func (p *Javascript) IntGetter() func() (int64, error) {
	return func() (res int64, err error) {
		if p.in != nil {
			err = transformGetter(p)
		}
		if err == nil {
			var v otto.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				res, err = v.ToInteger()
			}
		}

		return res, err
	}
}

// StringGetter parses string from request
func (p *Javascript) StringGetter() func() (string, error) {
	return func() (res string, err error) {
		if p.in != nil {
			err = transformGetter(p)
		}
		if err == nil {
			var v otto.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				res, err = v.ToString()
			}
		}

		return res, err
	}
}

// BoolGetter parses bool from request
func (p *Javascript) BoolGetter() func() (bool, error) {
	return func() (res bool, err error) {
		if p.in != nil {
			err = transformGetter(p)
		}
		if err == nil {
			var v otto.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				res, err = v.ToBoolean()
			}
		}

		return res, err
	}
}

func (p *Javascript) paramAndEval(param string, val any) error {
	err := p.vm.Set(param, val)
	if err == nil {
		err = p.vm.Set("param", param)
	}
	if err == nil {
		err = p.vm.Set("val", val)
	}
	if err == nil {
		var v otto.Value
		v, err = p.vm.Eval(p.script)
		if err == nil && p.out != nil {
			err = transformSetter(p, v)
		}
	}
	return err
}

// IntSetter sends int request
func (p *Javascript) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		return p.paramAndEval(param, val)
	}
}

// FloatSetter sends int request
func (p *Javascript) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		return p.paramAndEval(param, val)
	}
}

// StringSetter sends string request
func (p *Javascript) StringSetter(param string) func(string) error {
	return func(val string) error {
		return p.paramAndEval(param, val)
	}
}

// BoolSetter sends bool request
func (p *Javascript) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		return p.paramAndEval(param, val)
	}
}

func transformGetter(p *Javascript) error {
	for _, cc := range p.in {
		name := cc.Name
		var val any
		if cc.Type == "bool" {
			f, err := NewBoolGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else if cc.Type == "int" {
			f, err := NewIntGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else if cc.Type == "float" {
			f, err := NewFloatGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else {
			f, err := NewStringGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		err := p.vm.Set(name, val)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func transformSetter(p *Javascript, v otto.Value) error {
	for _, cc := range p.out {
		name := cc.Name
		if cc.Type == "bool" {
			f, err := NewBoolSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			s, err := v.ToBoolean()
			if err == nil {
				err = f(s)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else if cc.Type == "int" {
			f, err := NewIntSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			s, err := v.ToInteger()
			if err == nil {
				err = f(s)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else if cc.Type == "float" {
			f, err := NewFloatSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			s, err := v.ToFloat()
			if err == nil {
				err = f(s)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else {
			f, err := NewStringSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			s, err := v.ToString()
			if err == nil {
				err = f(s)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}
