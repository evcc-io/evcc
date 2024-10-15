package provider

import (
	"context"

	"github.com/evcc-io/evcc/provider/javascript"
	"github.com/evcc-io/evcc/util"
	"github.com/robertkrimen/otto"
	"github.com/spf13/cast"
)

// Javascript implements Javascript request provider
type Javascript struct {
	vm     *otto.Otto
	script string
	in     []inputTransformation
	out    []outputTransformation
}

func init() {
	registry.AddCtx("js", NewJavascriptProviderFromConfig)
}

// NewJavascriptProviderFromConfig creates a Javascript provider
func NewJavascriptProviderFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		VM     string
		Script string
		In     []transformationConfig
		Out    []transformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := javascript.RegisteredVM(cc.VM, "")
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

	p := &Javascript{
		vm:     vm,
		script: cc.Script,
		in:     in,
		out:    out,
	}

	return p, nil
}

var _ FloatProvider = (*Javascript)(nil)

// FloatGetter parses float from request
func (p *Javascript) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		v, err := p.handleGetter()
		if err != nil {
			return 0, err
		}

		return cast.ToFloat64E(v)
	}, nil
}

var _ IntProvider = (*Javascript)(nil)

// IntGetter parses int64 from request
func (p *Javascript) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		v, err := p.handleGetter()
		if err != nil {
			return 0, err
		}

		return cast.ToInt64E(v)
	}, nil
}

var _ StringProvider = (*Javascript)(nil)

// StringGetter parses string from request
func (p *Javascript) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		v, err := p.handleGetter()
		if err != nil {
			return "", err
		}

		return cast.ToStringE(v)
	}, nil
}

var _ BoolProvider = (*Javascript)(nil)

// BoolGetter parses bool from request
func (p *Javascript) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		v, err := p.handleGetter()
		if err != nil {
			return false, err
		}

		return cast.ToBoolE(v)
	}, nil
}

func (p *Javascript) handleGetter() (any, error) {
	if err := transformInputs(p.in, p.setParamSync); err != nil {
		return nil, err
	}

	javascript.Lock()
	defer javascript.Unlock()

	return p.evaluate()
}

func (p *Javascript) handleSetter(param string, val any) error {
	if err := transformInputs(p.in, p.setParamSync); err != nil {
		return err
	}

	javascript.Lock()
	if err := p.setParam(param, val); err != nil {
		javascript.Unlock()
		return err
	}

	v, err := p.evaluate()
	if err != nil {
		javascript.Unlock()
		return err
	}

	javascript.Unlock()
	return transformOutputs(p.out, v)
}

func (p *Javascript) evaluate() (any, error) {
	v, err := p.vm.Eval(p.script)
	if err != nil {
		return nil, err
	}

	vv, err := v.Export()
	if err != nil {
		return nil, err
	}

	if vv == nil {
		return nil, nil
	}

	return normalizeValue(vv)
}

func (p *Javascript) setParam(param string, val any) error {
	return p.vm.Set(param, val)
}

// setParamSync is the synchronized version of setParam
func (p *Javascript) setParamSync(param string, val any) error {
	javascript.Lock()
	defer javascript.Unlock()
	return p.setParam(param, val)
}

var _ SetIntProvider = (*Javascript)(nil)

// IntSetter sends int request
func (p *Javascript) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ SetFloatProvider = (*Javascript)(nil)

// FloatSetter sends float request
func (p *Javascript) FloatSetter(param string) (func(float64) error, error) {
	return func(val float64) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ SetStringProvider = (*Javascript)(nil)

// StringSetter sends string request
func (p *Javascript) StringSetter(param string) (func(string) error, error) {
	return func(val string) error {
		return p.handleSetter(param, val)
	}, nil
}

var _ SetBoolProvider = (*Javascript)(nil)

// BoolSetter sends bool request
func (p *Javascript) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return p.handleSetter(param, val)
	}, nil
}
