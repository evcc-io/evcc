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

		v, err := p.evaluate()
		if err != nil {
			return 0, err
		}

		vv, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("not a float: %s", v)
		}

		return vv, nil
	}
}

// IntGetter parses int64 from request
func (p *Go) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return 0, err
		}

		v, err := p.evaluate()
		if err != nil {
			return 0, err
		}

		vv, ok := v.(int64)
		if !ok {
			return 0, fmt.Errorf("not a int: %s", v)
		}

		return vv, nil
	}
}

// StringGetter parses string from request
func (p *Go) StringGetter() func() (string, error) {
	return func() (string, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return "", err
		}

		v, err := p.evaluate()
		if err != nil {
			return "", err
		}

		vv, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("not a string: %s", v)
		}

		return vv, nil
	}
}

// BoolGetter parses bool from request
func (p *Go) BoolGetter() func() (bool, error) {
	return func() (bool, error) {
		if err := handleInTransformation(p.in, p.setParam); err != nil {
			return false, err
		}

		v, err := p.evaluate()
		if err != nil {
			return false, err
		}

		vv, ok := v.(bool)
		if !ok {
			return false, fmt.Errorf("not a bool: %s", v)
		}

		return vv, nil
	}
}

func (p *Go) handleSetter(param string, val any) error {
	err := p.setParam(param, val)
	if err != nil {
		return err
	}

	vv, err2 := p.evaluate()
	if err2 != nil {
		return err2
	}

	return handleOutTransformation(p.out, vv)
}

func (p *Go) evaluate() (any, error) {
	v, err := p.vm.Eval(p.script)

	if err != nil {
		return nil, err
	}

	vv := v.Interface()

	switch t := vv.(type) {
	case int:
		return int64(vv.(int)), nil
	case int32:
		return int64(vv.(int32)), nil
	case float32:
		return float64(vv.(float32)), nil
	case int64:
	case float64:
	case bool:
	case string:
		return vv, nil
	default:
		return nil, fmt.Errorf("type not supported: %v", t)
	}

	return vv, nil
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
		return p.handleSetter(param, val)
	}
}

// FloatSetter sends float request
func (p *Go) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		return p.handleSetter(param, val)
	}
}

// StringSetter sends string request
func (p *Go) StringSetter(param string) func(string) error {
	return func(val string) error {
		return p.handleSetter(param, val)
	}
}

// BoolSetter sends bool request
func (p *Go) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		return p.handleSetter(param, val)
	}
}
