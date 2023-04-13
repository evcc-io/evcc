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
}

func init() {
	registry.Add("js", NewJavascriptProviderFromConfig)
}

// NewJavascriptProviderFromConfig creates a Javascript provider
func NewJavascriptProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		VM     string
		Script string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	vm, err := javascript.RegisteredVM(cc.VM, "")
	if err != nil {
		return nil, err
	}

	p := &Javascript{
		vm:     vm,
		script: cc.Script,
	}

	return p, nil
}

// FloatGetter parses float from request
func (p *Javascript) FloatGetter() func() (float64, error) {
	return func() (res float64, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			res, err = v.ToFloat()
		}

		return res, err
	}
}

// IntGetter parses int64 from request
func (p *Javascript) IntGetter() func() (int64, error) {
	return func() (res int64, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			res, err = v.ToInteger()
		}

		return res, err
	}
}

// StringGetter parses string from request
func (p *Javascript) StringGetter() func() (string, error) {
	return func() (res string, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			res, err = v.ToString()
		}

		return res, err
	}
}

// BoolGetter parses bool from request
func (p *Javascript) BoolGetter() func() (bool, error) {
	return func() (res bool, err error) {
		v, err := p.vm.Eval(p.script)
		if err == nil {
			res, err = v.ToBoolean()
		}

		return res, err
	}
}

func (p *Javascript) paramAndEval(param string, val any) error {
	err := p.vm.Set(param, val)
	if err == nil {
		err = p.vm.Set("param", param)
	}
	if err == nil {
		err = p.vm.Set("val", val)
	}
	if err == nil {
		_, err = p.vm.Eval(p.script)
	}
	return err
}

// IntSetter sends int request
func (p *Javascript) IntSetter(param string) func(int64) error {
	return func(val int64) error {
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
