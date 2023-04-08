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
}

func init() {
	registry.Add("go", NewGoProviderFromConfig)
}

// NewGoProviderFromConfig creates a Go provider
func NewGoProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		VM     string
		Script string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := golang.RegisteredVM(cc.VM, "")
	if err != nil {
		return nil, err
	}

	p := &Go{
		vm:     vm,
		script: cc.Script,
	}

	return p, nil
}

// FloatGetter parses float from request
func (p *Go) FloatGetter() func() (float64, error) {
	return func() (res float64, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if typ := reflect.TypeOf(res); v.CanConvert(typ) {
				res = v.Convert(typ).Float()
			} else {
				err = fmt.Errorf("not a float: %v", v)
			}
		}

		return res, err
	}
}

// IntGetter parses int64 from request
func (p *Go) IntGetter() func() (int64, error) {
	return func() (res int64, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if typ := reflect.TypeOf(res); v.CanConvert(typ) {
				res = v.Convert(typ).Int()
			} else {
				err = fmt.Errorf("not an int: %v", v)
			}
		}

		return res, err
	}
}

// StringGetter parses string from request
func (p *Go) StringGetter() func() (string, error) {
	return func() (res string, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if typ := reflect.TypeOf(res); v.CanConvert(typ) {
				res = v.Convert(typ).String()
			} else {
				err = fmt.Errorf("not a string: %v", v)
			}
		}

		return res, err
	}
}

// BoolGetter parses bool from request
func (p *Go) BoolGetter() func() (bool, error) {
	return func() (res bool, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			if typ := reflect.TypeOf(res); v.CanConvert(typ) {
				res = v.Convert(typ).Bool()
			} else {
				err = fmt.Errorf("not a boolean: %v", v)
			}
		}

		return res, err
	}
}

func (p *Go) paramAndEval(param string, val any) error {
	_, err := p.vm.Eval(fmt.Sprintf("%s := %v;", param, val))
	if err == nil {
		_, err = p.vm.Eval(p.script)
	}
	return err
}

func (p *Go) IntSetter(param string) func(int64) error {
	return func(val int64) error {
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
