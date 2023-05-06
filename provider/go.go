package provider

import (
	"fmt"
	"reflect"

	"github.com/evcc-io/evcc/provider/golang"
	"github.com/evcc-io/evcc/util"
	"github.com/traefik/yaegi/interp"
)

// Go implements Go request provider
type Go struct {
	vm     *interp.Interpreter
	script string
	in     []InTransformation
	out    []OutTransformation
}

func init() {
	registry.Add("go", NewGoProviderFromConfig)
}

// NewGoProviderFromConfig creates a Go provider
func NewGoProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		VM     string
		Script string
		In     []TransformationConfig
		Out    []TransformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := golang.RegisteredVM(cc.VM, "")
	if err != nil {
		return nil, err
	}

	in, err := ConvertInFunctions(cc.In)
	if err != nil {
		return nil, err
	}

	out, err := ConvertOutFunctions(cc.Out)
	if err != nil {
		return nil, err
	}

	p := &Go{
		vm:     vm,
		script: cc.Script,
		in:     in,
		out:    out,
	}

	return p, nil
}

// FloatGetter parses float from request
func (p *Go) FloatGetter() func() (float64, error) {
	return func() (res float64, err error) {
		if p.in != nil {
			err = p.transformGetter()
		}
		if err == nil {
			var v reflect.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				if typ := reflect.TypeOf(res); v.CanConvert(typ) {
					res = v.Convert(typ).Float()
				} else {
					err = fmt.Errorf("not a float: %v", v)
				}
			}
		}
		return res, err
	}
}

// IntGetter parses int64 from request
func (p *Go) IntGetter() func() (int64, error) {
	return func() (res int64, err error) {

		if p.in != nil {
			err = p.transformGetter()
		}
		if err == nil {
			var v reflect.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				if typ := reflect.TypeOf(res); v.CanConvert(typ) {
					res = v.Convert(typ).Int()
				} else {
					err = fmt.Errorf("not an int: %v", v)
				}
			}
		}

		return res, err
	}
}

// StringGetter parses string from request
func (p *Go) StringGetter() func() (string, error) {
	return func() (res string, err error) {
		if p.in != nil {
			err = p.transformGetter()
		}
		if err == nil {
			var v reflect.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				if typ := reflect.TypeOf(res); v.CanConvert(typ) {
					res = v.Convert(typ).String()
				} else {
					err = fmt.Errorf("not a string: %v", v)
				}
			}
		}
		return res, err
	}
}

// BoolGetter parses bool from request
func (p *Go) BoolGetter() func() (bool, error) {
	return func() (res bool, err error) {
		if p.in != nil {
			err = p.transformGetter()
		}
		if err == nil {
			var v reflect.Value
			v, err = p.vm.Eval(p.script)
			if err == nil {
				if typ := reflect.TypeOf(res); v.CanConvert(typ) {
					res = v.Convert(typ).Bool()
				} else {
					err = fmt.Errorf("not a boolean: %v", v)
				}
			}
		}

		return res, err
	}
}

func (p *Go) paramAndEval(param string, val any) error {
	if str, ok := val.(string); ok {
		val = "\"" + str + "\""
	}

	_, err := p.vm.Eval(fmt.Sprintf("%s := %v;", param, val))
	if err == nil {
		_, err = p.vm.Eval(fmt.Sprintf("param := %v;", param))
	}
	if err == nil {
		_, err = p.vm.Eval(fmt.Sprintf("val := %v;", val))
	}
	if err == nil {
		var v reflect.Value
		v, err = p.vm.Eval(p.script)
		if err == nil && p.out != nil {
			err = p.transformSetter(v)
		}
	}
	return err
}

// IntSetter sends int request
func (p *Go) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		return p.paramAndEval(param, val)
	}
}

// FloatSetter sends float request
func (p *Go) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		return p.paramAndEval(param, val)
	}
}

// StringSetter sends string request
func (p *Go) StringSetter(param string) func(string) error {
	return func(val string) error {
		return p.paramAndEval(param, val)
	}
}

// BoolSetter sends bool request
func (p *Go) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		return p.paramAndEval(param, val)
	}
}

func (p *Go) transformGetter() error {
	for _, cc := range p.in {
		val, err := cc.function()
		if err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}

		err = p.paramAndEval(cc.name, val)
		if err != nil {
			return fmt.Errorf("%s: %w", cc.name, err)
		}
	}
	return nil
}
func (p *Go) transformSetter(v reflect.Value) error {
	for _, cc := range p.out {
		name := cc.name
		if cc.Type == "bool" {
			var err error
			if v.CanConvert(reflect.TypeOf(true)) {
				err = cc.function(v.Convert(reflect.TypeOf(true)).Bool())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else if cc.Type == "int" {
			var err error
			if v.CanConvert(reflect.TypeOf(0)) {
				err = cc.function(v.Convert(reflect.TypeOf(0)).Int())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else if cc.Type == "float" {
			var err error
			if v.CanConvert(reflect.TypeOf(0.0)) {
				err = cc.function(v.Convert(reflect.TypeOf(0.0)).Float())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		} else {
			var err error
			if v.CanConvert(reflect.TypeOf("")) {
				err = cc.function(v.Convert(reflect.TypeOf("")).String())
			} else {
				err = fmt.Errorf("not a int: %s", v)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}
