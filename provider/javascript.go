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

// NewJavascriptProviderFromConfig creates a HTTP provider
func NewJavascriptProviderFromConfig(other map[string]interface{}) (*Javascript, error) {
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
func (p *Javascript) FloatGetter() (res float64, err error) {
	v, err := p.vm.Eval(p.script)
	if err == nil {
		res, err = v.ToFloat()
	}

	return res, err
}

// IntGetter parses int64 from request
func (p *Javascript) IntGetter() (res int64, err error) {
	v, err := p.vm.Eval(p.script)
	if err == nil {
		res, err = v.ToInteger()
	}

	return res, err
}

// StringGetter sends string request
func (p *Javascript) StringGetter() (res string, err error) {
	v, err := p.vm.Eval(p.script)
	if err == nil {
		res, err = v.ToString()
	}

	return res, err
}

// BoolGetter parses bool from request
func (p *Javascript) BoolGetter() (res bool, err error) {
	v, err := p.vm.Eval(p.script)
	if err == nil {
		res, err = v.ToBoolean()
	}

	return res, err
}

// // IntSetter sends int request
// func (p *Javascript) IntSetter(param int64) error {
// 	body := util.FormatValue(p.body, param)
// 	_, err := p.request(body)
// 	return err
// }

// // StringSetter sends string request
// func (p *Javascript) StringSetter(param string) error {
// 	body := util.FormatValue(p.body, param)
// 	_, err := p.request(body)
// 	return err
// }

// // BoolSetter sends bool request
// func (p *Javascript) BoolSetter(param bool) error {
// 	body := util.FormatValue(p.body, param)
// 	_, err := p.request(body)
// 	return err
// }
