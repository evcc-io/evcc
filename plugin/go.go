package plugin

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/plugin/golang"
	"github.com/evcc-io/evcc/util"
	"github.com/traefik/yaegi/interp"
)

// Go implements Go request provider
type Go struct {
	vm     func() (*interp.Interpreter, error)
	script string
	in     []inputTransformation
	out    []outputTransformation
}

func init() {
	registry.AddCtx("go", NewGoPluginFromConfig)
}

// NewGoPluginFromConfig creates a Go provider
func NewGoPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		VM     string
		Script string
		In     []transformationConfig
		Out    []transformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	_, err := golang.RegisteredVM(cc.VM, "")
	if err != nil {
		return nil, err
	}

	in, err := configureInputs(ctx, cc.In)
	if err != nil {
		return nil, err
	}

	out, err := configureOutputs(ctx, cc.Out)
	if err != nil {
		return nil, err
	}

	p := &Go{
		// recreate VM on each invocation
		vm:     func() (*interp.Interpreter, error) { return golang.RegisteredVM(cc.VM, "") },
		script: cc.Script,
		in:     in,
		out:    out,
	}

	return p, nil
}

var _ FloatGetter = (*Go)(nil)

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

var _ IntGetter = (*Go)(nil)

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

var _ StringGetter = (*Go)(nil)

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

var _ BoolGetter = (*Go)(nil)

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
	vm, err := p.vm()
	if err != nil {
		return nil, err
	}

	if err := transformInputs(p.in, p.setParam(vm)); err != nil {
		return nil, err
	}

	return p.evaluate(vm)
}

func (p *Go) handleSetter(param string, val any) error {
	vm, err := p.vm()
	if err != nil {
		return err
	}

	setParam := p.setParam(vm)

	if err := transformInputs(p.in, setParam); err != nil {
		return err
	}

	if err := setParam(param, val); err != nil {
		return err
	}

	vv, err := p.evaluate(vm)
	if err != nil {
		return err
	}

	return transformOutputs(p.out, vv)
}

func (p *Go) evaluate(vm *interp.Interpreter) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		err = backoff.Permanent(err)
	}()

	v, err := vm.Eval(p.script)
	if err != nil {
		return nil, err
	}

	if !v.IsValid() {
		return nil, errors.New("missing result")
	}

	if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && v.IsNil() {
		return nil, nil
	}

	return normalizeValue(v.Interface())
}

func (p *Go) setParam(vm *interp.Interpreter) func(param string, val any) error {
	return func(param string, val any) error {
		_, err := vm.Eval(fmt.Sprintf("%s := %#v;", param, val))
		return err
	}
}

var _ IntSetter = (*Go)(nil)

// IntSetter sends int request
func (p *Go) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ FloatSetter = (*Go)(nil)

// FloatSetter sends float request
func (p *Go) FloatSetter(param string) (func(float64) error, error) {
	return func(val float64) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ StringSetter = (*Go)(nil)

// StringSetter sends string request
func (p *Go) StringSetter(param string) (func(string) error, error) {
	return func(val string) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ BoolSetter = (*Go)(nil)

// BoolSetter sends bool request
func (p *Go) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return p.handleSetter(param, val)
	}, nil
}
