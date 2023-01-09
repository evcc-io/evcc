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
	set    *Config
}

func init() {
	registry.Add("js", NewJavascriptProviderFromConfig)
}

// NewJavascriptProviderFromConfig creates a HTTP provider
func NewJavascriptProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		VM     string
		Script string
		Get    []Config
		Set    *Config
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
		set:    cc.Set,
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

func (p *Javascript) setParam(param string, val interface{}) error {
	err := p.vm.Set(param, val)
	if err == nil {
		err = p.vm.Set("param", param)
	}
	if err == nil {
		err = p.vm.Set("val", val)
	}
	return err
}

// IntSetter sends int request
func (p *Javascript) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.set != nil {
				setTransformed(param, p, v)
			}
		}
		return err
	}
}

// FloatSetter sends float request
func (p *Javascript) FloatSetter(param string) func(float64) error {
	return func(val float64) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.set != nil {
				setTransformed(param, p, v)
			}
		}
		return err
	}
}

// StringSetter sends string request
func (p *Javascript) StringSetter(param string) func(string) error {
	return func(val string) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.set != nil {
				setTransformed(param, p, v)
			}
		}
		return err
	}
}

// BoolSetter sends bool request
func (p *Javascript) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		err := p.setParam(param, val)
		if err == nil {
			v, err := p.vm.Eval(p.script)
			if err == nil && p.set != nil {
				setTransformed(param, p, v)
			}
		}
		return err
	}
}

func setTransformed(param string, p *Javascript, v otto.Value) {
	if v.IsBoolean() {
		f, err := NewBoolSetterFromConfig(param, *p.set)
		if err == nil {
			s, err := v.ToBoolean()
			if err == nil {
				err = f(s)
			}
		}
	} else if v.IsNumber() {
		// TODO: how to detect ints?
		f, err := NewFloatSetterFromConfig(param, *p.set)
		if err == nil {
			s, err := v.ToFloat()
			if err == nil {
				err = f(s)
			}
		}
	} else {
		f, err := NewStringSetterFromConfig(param, *p.set)
		if err == nil {
			s, err := v.ToString()
			if err == nil {
				err = f(s)
			}
		}
	}
}
