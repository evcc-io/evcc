package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type calcProvider struct {
	add, mul, div, in []func() (float64, error)
	sign              func() (float64, error)
	calc              func([]float64) (float64, error)
}

func init() {
	registry.AddCtx("calc", NewCalcFromConfig)
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Add     []Config
		Mul     []Config
		Div     []Config
		Sign    *Config
		Formula string
		In      []Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	cnt := min(len(cc.Add), 1) + min(len(cc.Mul), 1) + min(len(cc.Div), 1) + min(len(cc.In), 1)
	if cc.Sign != nil {
		cnt++
	}
	if cnt != 1 {
		return nil, errors.New("can only have either add, mul, div, sign or formula")
	}

	o := &calcProvider{}

	for idx, cc := range cc.Add {
		f, err := NewFloatGetterFromConfig(ctx, cc)
		if err != nil {
			return nil, fmt.Errorf("add[%d]: %w", idx, err)
		}
		o.add = append(o.add, f)
	}

	for idx, cc := range cc.Mul {
		f, err := NewFloatGetterFromConfig(ctx, cc)
		if err != nil {
			return nil, fmt.Errorf("mul[%d]: %w", idx, err)
		}
		o.mul = append(o.mul, f)
	}

	for idx, cc := range cc.Div {
		f, err := NewFloatGetterFromConfig(ctx, cc)
		if err != nil {
			return nil, fmt.Errorf("div[%d]: %w", idx, err)
		}
		o.div = append(o.div, f)
	}

	if cc.Sign != nil {
		f, err := NewFloatGetterFromConfig(ctx, *cc.Sign)
		if err != nil {
			return nil, fmt.Errorf("sign: %w", err)
		}
		o.sign = f
	}

	for idx, cc := range cc.In {
		f, err := NewFloatGetterFromConfig(ctx, cc)
		if err != nil {
			return nil, fmt.Errorf("in[%d]: %w", idx, err)
		}
		o.in = append(o.in, f)
	}

	if len(o.in) > 0 {
		if err := o.program(cc.Formula); err != nil {
			return nil, err
		}
	}

	return o, nil
}

func (o *calcProvider) program(formula string) (err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	vm := interp.New(interp.Options{})
	if err := vm.Use(stdlib.Symbols); err != nil {
		return err
	}

	in := strings.Join(lo.Map(o.in, func(_ func() (float64, error), idx int) string {
		return fmt.Sprintf("in%d", idx)
	}), ", ")

	if _, err := vm.Eval(fmt.Sprintf(`
			import "math"
			var %s float64
		`, in)); err != nil {
		return err
	}

	prg, err := vm.Compile(formula)
	if err != nil {
		return err
	}

	o.calc = func(in []float64) (float64, error) {
		for idx, v := range in {
			if _, err := vm.Eval(fmt.Sprintf("in%d = %f", idx, v)); err != nil {
				return 0, err
			}
		}

		res, err := vm.Execute(prg)
		if err != nil {
			return 0, err
		}

		if !res.CanFloat() {
			return 0, errors.New("formula did not return a float value")
		}

		return res.Float(), nil
	}

	// test the formula
	_, err = o.calc(make([]float64, len(o.in)))
	return err
}

var _ IntProvider = (*calcProvider)(nil)

func (o *calcProvider) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		f, err := o.floatGetter()
		return int64(f), err
	}, nil
}

var _ StringProvider = (*calcProvider)(nil)

func (o *calcProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		f, err := o.floatGetter()
		return fmt.Sprintf("%c", int(f)), err
	}, nil
}

var _ FloatProvider = (*calcProvider)(nil)

func (o *calcProvider) FloatGetter() (func() (float64, error), error) {
	return o.floatGetter, nil
}

func (o *calcProvider) floatGetter() (float64, error) {
	var res float64

	switch {
	case len(o.in) > 0:
		val := make([]float64, len(o.in))
		for idx, p := range o.in {
			v, err := p()
			if err != nil {
				return 0, fmt.Errorf("in[%d]: %w", idx, err)
			}
			val[idx] = v
		}
		return o.calc(val)

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

	case len(o.div) > 0:
		for idx, p := range o.div {
			v, err := p()
			if err != nil {
				return 0, fmt.Errorf("div[%d]: %w", idx, err)
			}
			if idx == 0 {
				if v == 0 {
					break
				}
				res = v
			} else if v != 0 {
				res /= v
			} else if v == 0 {
				res = 0
				break
			}
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
