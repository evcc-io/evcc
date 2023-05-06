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
func NewGoProviderFromConfig(other map[string]interface{}) (Provider, error) {
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
				res, err = p.convertToFloat(v)
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
				res, err = p.convertToInt(v)
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
				res, err = p.convertToString(v)
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
				res, err = p.convertToBool(v)
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
	return transformGetter(p, p.in)
}

func (p *Go) transformSetter(v reflect.Value) error {
	return transformSetter(p, p.out, v)
}

func (p *Go) convertToInt(v reflect.Value) (int64, error) {
	if v.CanConvert(reflect.TypeOf(0)) {
		return v.Convert(reflect.TypeOf(0)).Int(), nil
	} else {
		return 0, fmt.Errorf("not a int: %s", v)
	}
}

func (p *Go) convertToString(v reflect.Value) (string, error) {
	if v.CanConvert(reflect.TypeOf("")) {
		return v.Convert(reflect.TypeOf("")).String(), nil
	} else {
		return "", fmt.Errorf("not a string: %s", v)
	}
}

func (p *Go) convertToFloat(v reflect.Value) (float64, error) {
	if v.CanConvert(reflect.TypeOf(0.0)) {
		return v.Convert(reflect.TypeOf(0.0)).Float(), nil
	} else {
		return 0.0, fmt.Errorf("not a float: %s", v)
	}
}

func (p *Go) convertToBool(v reflect.Value) (bool, error) {
	if v.CanConvert(reflect.TypeOf(true)) {
		return v.Convert(reflect.TypeOf(true)).Bool(), nil
	} else {
		return false, fmt.Errorf("not a bool: %s", v)
	}
}
