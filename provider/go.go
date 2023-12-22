package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/provider/golang"
	"github.com/evcc-io/evcc/util"
	"github.com/traefik/yaegi/interp"
)

// Go implements Go request provider
type Go struct {
	vm     *interp.Interpreter
	script string
	in     []inputTransformation
	out    []outputTransformation
}

func init() {
	registry.Add("go", NewGoProviderFromConfig)
}

// NewGoProviderFromConfig creates a Go provider
func NewGoProviderFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		VM     string
		Script string
		In     []transformationConfig
		Out    []transformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := golang.RegisteredVM(cc.VM, "")
	if err != nil {
		return nil, err
	}

	in, err := configureInputs(cc.In)
	if err != nil {
		return nil, err
	}

	out, err := configureOutputs(cc.Out)
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

var _ FloatProvider = (*Go)(nil)

// FloatGetter parses float from request
func (p *Go) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		v, err := p.handleGetter()
		if err != nil {
			return 0, err
		}

		vv, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("not a float: %v", v)
		}

		return vv, nil
	}, nil
}

var _ IntProvider = (*Go)(nil)

// IntGetter parses int64 from request
func (p *Go) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		v, err := p.handleGetter()
		if err != nil {
			return 0, err
		}

		vv, ok := v.(int64)
		if !ok {
			return 0, fmt.Errorf("not a int: %v", v)
		}

		return vv, nil
	}, nil
}

var _ StringProvider = (*Go)(nil)

// StringGetter parses string from request
func (p *Go) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		v, err := p.handleGetter()
		if err != nil {
			return "", err
		}

		vv, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("not a string: %v", v)
		}

		return vv, nil
	}, nil
}

var _ BoolProvider = (*Go)(nil)

// BoolGetter parses bool from request
func (p *Go) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		v, err := p.handleGetter()
		if err != nil {
			return false, err
		}

		vv, ok := v.(bool)
		if !ok {
			return false, fmt.Errorf("not a bool: %v", v)
		}

		return vv, nil
	}, nil
}

func (p *Go) handleGetter() (any, error) {
	if err := transformInputs(p.in, p.setParam); err != nil {
		return nil, err
	}

	return p.evaluate()
}

func (p *Go) handleSetter(param string, val any) error {
	if err := p.setParam(param, val); err != nil {
		return err
	}

	vv, err := p.evaluate()
	if err != nil {
		return err
	}

	return transformOutputs(p.out, vv)
}

func (p *Go) evaluate() (any, error) {
	v, err := p.vm.Eval(p.script)
	if err != nil {
		return nil, err
	}

	return normalizeValue(v.Interface())
}

func (p *Go) setParam(param string, val any) error {
	if str, ok := val.(string); ok {
		val = "\"" + str + "\""
	}

	_, err := p.vm.Eval(fmt.Sprintf("%s := %v;", param, val))
	return err
}

var _ SetIntProvider = (*Go)(nil)

// IntSetter sends int request
func (p *Go) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ SetFloatProvider = (*Go)(nil)

// FloatSetter sends float request
func (p *Go) FloatSetter(param string) (func(float64) error, error) {
	return func(val float64) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ SetStringProvider = (*Go)(nil)

// StringSetter sends string request
func (p *Go) StringSetter(param string) (func(string) error, error) {
	return func(val string) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ SetBoolProvider = (*Go)(nil)

// BoolSetter sends bool request
func (p *Go) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return p.handleSetter(param, val)
	}, nil
}
