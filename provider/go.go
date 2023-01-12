package provider

import (
	"fmt"
	"github.com/evcc-io/evcc/util"
	"github.com/traefik/yaegi/interp"
	"reflect"
)

// Go implements Go request provider
type Go struct {
	vm        *interp.Interpreter
	script    string
	transform []TransformationConfig
}

//type TransformationConfig struct {
//	Name, Type string
//	Config     Config
//}

func init() {
	registry.Add("go", NewGoProviderFromConfig)
}

// NewGoProviderFromConfig creates a Go provider
func NewGoProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		Script    string
		Transform []TransformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := &Go{
		vm:        interp.New(interp.Options{}),
		script:    cc.Script,
		transform: cc.Transform,
	}

	return p, nil
}

// FloatGetter parses float from request
func (p *Go) FloatGetter() func() (float64, error) {
	return func() (res float64, err error) {
		if p.transform != nil {
			err = transformGetterGo(p)
		}
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil {
				if v.CanConvert(reflect.TypeOf(0.0)) {
					res = v.Convert(reflect.TypeOf(0.0)).Float()
				} else {
					err = fmt.Errorf("not a float: %s", v)
				}
			}
		}
		return res, err
	}
}

// IntGetter parses int64 from request
func (p *Go) IntGetter() func() (int64, error) {
	return func() (res int64, err error) {
		if p.transform != nil {
			err = transformGetterGo(p)
		}
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if v.CanConvert(reflect.TypeOf(0)) {
				res = v.Convert(reflect.TypeOf(0)).Int()
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
		}

		return res, err
	}
}

// StringGetter sends string request
func (p *Go) StringGetter() func() (string, error) {
	return func() (res string, err error) {
		if p.transform != nil {
			err = transformGetterGo(p)
		}
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if v.CanConvert(reflect.TypeOf("")) {
				res = v.Convert(reflect.TypeOf("")).String()
			} else {
				err = fmt.Errorf("not a string: %s", v)
			}
		}

		return res, err
	}
}

// BoolGetter parses bool from request
func (p *Go) BoolGetter() func() (bool, error) {
	return func() (res bool, err error) {
		if p.transform != nil {
			err = transformGetterGo(p)
		}
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if v.CanConvert(reflect.TypeOf(true)) {
				res = v.Convert(reflect.TypeOf(true)).Bool()
			} else {
				err = fmt.Errorf("not a float: %s", v)
			}
		}

		return res, err
	}
}

func (p *Go) setParam(param string, val interface{}) error {
	_, err := p.vm.Eval(fmt.Sprintf("%s := %v;", param, val))
	return err
}

// IntSetter sends int request
func (p *Go) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.transform != nil {
				err = transformSetterGo(p.transform, v)
			}
		}
		return err
	}
}

// FloatSetter sends float request
func (p *Go) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		err := p.setParam(param, val)
		if err == nil {
			var v reflect.Value
			v, err = p.vm.Eval(p.script)
			if err == nil && p.transform != nil {
				err = transformSetterGo(p.transform, v)
			}
		}
		return err
	}
}

// StringSetter sends string request
func (p *Go) StringSetter(param string) func(string) error {
	return func(val string) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.transform != nil {
				err = transformSetterGo(p.transform, v)
			}
		}
		return err
	}
}

// BoolSetter sends bool request
func (p *Go) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.transform != nil {
				err = transformSetterGo(p.transform, v)
			}
		}
		return err
	}
}

func transformGetterGo(p *Go) error {
	for _, cc := range p.transform {
		name := cc.Name
		var val any
		if cc.Type == "bool" {
			f, err := NewBoolGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
		} else if cc.Type == "int" {
			f, err := NewIntGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
		} else if cc.Type == "float" {
			f, err := NewFloatGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
		} else {
			f, err := NewStringGetterFromConfig(cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			val, err = f()
		}
		err := p.setParam(name, val)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
func transformSetterGo(transforms []TransformationConfig, v reflect.Value) error {
	for _, cc := range transforms {
		name := cc.Name
		if cc.Type == "bool" {
			f, err := NewBoolSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if v.CanConvert(reflect.TypeOf(true)) {
				err = f(v.Convert(reflect.TypeOf(true)).Bool())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
		} else if cc.Type == "int" {
			f, err := NewIntSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if v.CanConvert(reflect.TypeOf(0)) {
				err = f(v.Convert(reflect.TypeOf(0)).Int())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
		} else if cc.Type == "float" {
			f, err := NewFloatSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if v.CanConvert(reflect.TypeOf(0.0)) {
				err = f(v.Convert(reflect.TypeOf(0.0)).Float())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
		} else {
			f, err := NewStringSetterFromConfig(name, cc.Config)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if v.CanConvert(reflect.TypeOf("")) {
				err = f(v.Convert(reflect.TypeOf("")).String())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
		}
	}
	return nil
}
