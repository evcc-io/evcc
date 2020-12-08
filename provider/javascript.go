package provider

import (
	"github.com/andig/evcc/util"
	"github.com/robertkrimen/otto"
)

// Javascript implements Javascript request provider
type Javascript struct {
	log    *util.Logger
	vm     *otto.Otto
	script string
}

func init() {
	registry.Add("js", NewJavascriptProviderFromConfig)
}

// NewJavascriptProviderFromConfig creates a HTTP provider
func NewJavascriptProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		Script string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("js")

	p := &Javascript{
		log:    log,
		vm:     otto.New(),
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

// StringGetter sends string request
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

// IntSetter sends int request
func (p *Javascript) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		err := p.vm.Set(param, val)
		if err == nil {
			_, err = p.vm.Eval(p.script)
		}
		return err
	}
}

// StringSetter sends string request
func (p *Javascript) StringSetter(param string) func(string) error {
	return func(val string) error {
		err := p.vm.Set(param, val)
		if err == nil {
			_, err = p.vm.Eval(p.script)
		}
		return err
	}
}

// BoolSetter sends bool request
func (p *Javascript) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		err := p.vm.Set(param, val)
		if err == nil {
			_, err = p.vm.Eval(p.script)
		}
		return err
	}
}
