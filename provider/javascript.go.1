package provider

import (
	"github.com/evcc-io/evcc/provider/javascript"
	"github.com/evcc-io/evcc/util"
	"github.com/robertkrimen/otto"
)

// Javascript implements Javascript request provider
type Javascript struct {
	vm     *otto.Otto
	script string
	in     []InTransformation
	out    []OutTransformation
}

func init() {
	registry.Add("js", NewJavascriptProviderFromConfig)
}

// NewJavascriptProviderFromConfig creates a Javascript provider
func NewJavascriptProviderFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		VM     string
		Script string
		In     []TransformationConfig
		Out    []TransformationConfig
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := javascript.RegisteredVM(cc.VM, "")
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

	p := &Javascript{
		vm:     vm,
		script: cc.Script,
		in:     in,
		out:    out,
	}

	return p, nil
}

// FloatGetter parses float from request
func (p *Javascript) FloatGetter() func() (float64, error) {
	return func() (float64, error) {
		if err := handleInTransformation(p); err != nil {
			return 0, err
		}

		v, err := p.evaluate()
		if err != nil {
			return 0, err
		}

		return p.convertToFloat(v)
	}
}

// IntGetter parses int64 from request
func (p *Javascript) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		if err := handleInTransformation(p); err != nil {
			return 0, err
		}

		v, err := p.evaluate()
		if err != nil {
			return 0, err
		}

		return p.convertToInt(v)
	}
}

// StringGetter parses string from request
func (p *Javascript) StringGetter() func() (string, error) {
	return func() (string, error) {
		if err := handleInTransformation(p); err != nil {
			return "", err
		}

		v, err := p.evaluate()
		if err != nil {
			return "", err
		}

		return p.convertToString(v)
	}
}

// BoolGetter parses bool from request
func (p *Javascript) BoolGetter() func() (bool, error) {
	return func() (bool, error) {
		if err := handleInTransformation(p); err != nil {
			return false, err
		}

		v, err := p.evaluate()
		if err != nil {
			return false, err
		}

		return p.convertToBool(v)
	}
}

func (p *Javascript) paramAndEval(param string, val any) error {
	err := p.setParam(param, val)
	if err != nil {
		return err
	}

	v, err := p.evaluate()
	if err != nil {
		return err
	}

	return handleOutTransformation(p, v)
}

func (p *Javascript) setParam(param string, val any) error {
	return p.vm.Set(param, val)
}

func (p *Javascript) evaluate() (otto.Value, error) {
	return p.vm.Eval(p.script)
}

// IntSetter sends int request
func (p *Javascript) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		return p.paramAndEval(param, val)
	}
}

// FloatSetter sends float request
func (p *Javascript) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		return p.paramAndEval(param, val)
	}
}

// StringSetter sends string request
func (p *Javascript) StringSetter(param string) func(string) error {
	return func(val string) error {
		return p.paramAndEval(param, val)
	}
}

// BoolSetter sends bool request
func (p *Javascript) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		return p.paramAndEval(param, val)
	}
}

func (p *Javascript) convertToInt(v otto.Value) (int64, error) {
	return v.ToInteger()
}

func (p *Javascript) convertToString(v otto.Value) (string, error) {
	return v.ToString()
}

func (p *Javascript) convertToFloat(v otto.Value) (float64, error) {
	return v.ToFloat()
}

func (p *Javascript) convertToBool(v otto.Value) (bool, error) {
	return v.ToBoolean()
}

func (p *Javascript) inTransformations() []InTransformation {
	return p.in
}

func (p *Javascript) outTransformations() []OutTransformation { //nolint:golint,unused
	return p.out
}
