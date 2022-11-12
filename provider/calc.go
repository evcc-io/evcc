package provider

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type calcProvider struct {
	add, mul []func() (float64, error)
	sign     func() (float64, error)
}

func init() {
	registry.Add("calc", NewCalcFromConfig)
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		Add  []Config
		Mul  []Config
		Sign *Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &calcProvider{}

	if len(cc.Add) > 0 && len(cc.Mul) > 0 || len(cc.Add) > 0 && cc.Sign != nil || len(cc.Mul) > 0 && cc.Sign != nil {
		return nil, errors.New("can only have either add, mul or sign")
	}

	for idx, cc := range cc.Add {
		f, err := NewFloatGetterFromConfig(cc)
		if err != nil {
			return nil, fmt.Errorf("add[%d]: %w", idx, err)
		}
		o.add = append(o.add, f)
	}

	for idx, cc := range cc.Mul {
		f, err := NewFloatGetterFromConfig(cc)
		if err != nil {
			return nil, fmt.Errorf("mul[%d]: %w", idx, err)
		}
		o.mul = append(o.mul, f)
	}

	if cc.Sign != nil {
		f, err := NewFloatGetterFromConfig(*cc.Sign)
		if err != nil {
			return nil, fmt.Errorf("sign: %w", err)
		}
		o.sign = f
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
	var res float64

	switch {
	case len(o.mul) > 0:
		res = 1
		for idx, p := range o.mul {
			v, err := p()
			if err != nil {
				return 0, fmt.Errorf("mul[%d]: %w", idx, err)
			}
			res *= v
		}

	case len(o.add) > 0:
		for idx, p := range o.add {
			v, err := p()
			if err != nil {
				return 0, fmt.Errorf("add[%d]: %w", idx, err)
			}
			res += v
		}

	default:
		v, err := o.sign()
		if err != nil {
			return 0, fmt.Errorf("sign: %w", err)
		}
		res = map[bool]float64{false: -1, true: 1}[v >= 0]
	}

	return res, nil
}
