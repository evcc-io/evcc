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
	return func() (float64, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return 0, err
		}

		v, err := p.vm.Eval(p.script)
		if err != nil {
			return 0, err
		}

		if typ := reflect.TypeOf(0.0); v.CanConvert(typ) {
			return v.Convert(typ).Float(), nil
		}

		return 0, fmt.Errorf("not a string: %s", v)
	}
}

// IntGetter parses int64 from request
func (p *Go) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return 0, err
		}

		v, err := p.vm.Eval(p.script)
		if err != nil {
			return 0, err
		}

		if typ := reflect.TypeOf(0); v.CanConvert(typ) {
			return v.Convert(typ).Int(), nil
		}

		return 0, fmt.Errorf("not a string: %s", v)
	}
}

// StringGetter parses string from request
func (p *Go) StringGetter() func() (string, error) {
	return func() (string, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return "", err
		}

		v, err := p.vm.Eval(p.script)
		if err != nil {
			return "", err
		}

		if typ := reflect.TypeOf(""); v.CanConvert(typ) {
			return v.Convert(typ).String(), nil
		}

		return "", fmt.Errorf("not a string: %s", v)
	}
}

// BoolGetter parses bool from request
func (p *Go) BoolGetter() func() (bool, error) {
	return func() (bool, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return false, err
		}

		v, err := p.vm.Eval(p.script)
		if err != nil {
			return false, err
		}

		if typ := reflect.TypeOf(false); v.CanConvert(typ) {
			return v.Convert(typ).Bool(), nil
		}

		return false, fmt.Errorf("not a string: %s", v)
	}
}

func (p *Go) paramAndEval(param string, val any) (*reflect.Value, error) {
	err := p.setParam(param, val)
	if err != nil {
		return nil, err
	}

	v, err := p.vm.Eval(p.script)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (p *Go) setParam(param string, val any) error {
	if str, ok := val.(string); ok {
		val = "\"" + str + "\""
	}

	_, err := p.vm.Eval(fmt.Sprintf("%s := %v;", param, val))
	return err
}

// IntSetter sends int request
func (p *Go) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		v, err := p.paramAndEval(param, val)
		if err != nil {
			return err
		}

		typ := reflect.TypeOf(0)
		if !v.CanConvert(typ) {
			return fmt.Errorf("not a int: %v", v)
		}

		return handleOutTransformation(p.out, v.Convert(typ).Int())
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
