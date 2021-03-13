package provider

import (
	"fmt"

	"github.com/andig/evcc/util"
)

type calcProvider struct {
	add []func() (float64, error)
}

func init() {
	registry.Add("calc", NewCalcFromConfig)
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		Add []Config
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &calcProvider{}

	for idx, cc := range cc.Add {
		f, err := NewFloatGetterFromConfig(cc)
		if err != nil {
			return nil, fmt.Errorf("add[%d]: %w", idx, err)
		}
		o.add = append(o.add, f)
	}

	return o, nil
}

func (o *calcProvider) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		f, err := o.floatGetter()
		return int64(f), err
	}
}

func (o *calcProvider) StringGetter() func() (string, error) {
	return func() (string, error) {
		f, err := o.floatGetter()
		return fmt.Sprintf("%c", int(f)), err
	}
}

func (o *calcProvider) FloatGetter() func() (float64, error) {
	return o.floatGetter
}

func (o *calcProvider) floatGetter() (float64, error) {
	var sum float64
	for idx, p := range o.add {
		v, err := p()
		if err != nil {
			return 0, fmt.Errorf("add[%d]: %w", idx, err)
		}
		sum += v
	}

	return sum, nil
}
