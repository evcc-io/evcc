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
	vm     *golang.VM
	script string
	in     []inputTransformation
	out    []outputTransformation
	prg    map[string]*interp.Program // compiled script by setter parameter
}

type goParam struct {
	name string
	val  any
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

	// the script is compiled once, repeated Eval leaks memory (https://github.com/traefik/yaegi/issues/1678)
	vm, err := golang.RegisteredVM(cc.VM, "")
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
		vm:     vm,
		script: cc.Script,
		in:     in,
		out:    out,
		prg:    make(map[string]*interp.Program),
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

// inputs reads the input values outside the interpreter lock
func (p *Go) inputs() ([]goParam, error) {
	var res []goParam
	err := transformInputs(p.in, func(name string, val any) error {
		res = append(res, goParam{name, val})
		return nil
	})
	return res, err
}

func (p *Go) handleGetter() (any, error) {
	params, err := p.inputs()
	if err != nil {
		return nil, err
	}

	return p.evaluate("", params)
}

func (p *Go) handleSetter(param string, val any) error {
	params, err := p.inputs()
	if err != nil {
		return err
	}

	vv, err := p.evaluate(param, append(params, goParam{param, val}))
	if err != nil {
		return err
	}

	return transformOutputs(p.out, vv)
}

// goType is the interpreter type of a parameter. Integers have always been
// handed to scripts as untyped constants, i.e. int.
func goType(val any) reflect.Type {
	if _, ok := val.(int64); ok {
		return reflect.TypeFor[int]()
	}
	return reflect.TypeOf(val)
}

// declare declares the parameters as globals
func (p *Go) declare(params []goParam) error {
	g := p.vm.Globals()

	for _, param := range params {
		typ := goType(param.val)

		// parameters of the same name share one global on a shared VM
		if v, ok := g[param.name]; ok {
			if v.Type() != typ {
				return fmt.Errorf("%s: type %s conflicts with %s", param.name, typ, v.Type())
			}
			continue
		}

		if _, err := p.vm.Eval(fmt.Sprintf("var %s %s", param.name, typ)); err != nil {
			return err
		}
	}

	return nil
}

func (p *Go) evaluate(key string, params []goParam) (res any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		err = backoff.Permanent(err)
	}()

	// named VMs are shared between plugins
	p.vm.Lock()
	defer p.vm.Unlock()

	prg, ok := p.prg[key]
	if !ok {
		if err := p.declare(params); err != nil {
			return nil, err
		}
	}

	// Globals panics between Compile and Execute of a script declaring new variables
	g := p.vm.Globals()

	if !ok {
		if prg, err = p.vm.Compile(p.script); err != nil {
			return nil, err
		}
		p.prg[key] = prg
	}

	for _, param := range params {
		v := g[param.name]
		v.Set(reflect.ValueOf(param.val).Convert(v.Type()))
	}

	v, err := p.vm.Execute(prg)
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
