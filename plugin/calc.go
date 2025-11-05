package plugin

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/evcc-io/evcc/util"
)

type calcPlugin struct {
	add, mul, div, min, max []func() (float64, error)
	abs, sign               func() (float64, error)
}

func init() {
	registry.AddCtx("calc", NewCalcFromConfig)
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Add  []Config
		Mul  []Config
		Div  []Config
		Min  []Config
		Max  []Config
		Abs  *Config
		Sign *Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	cnt := min(len(cc.Add), 1) + min(len(cc.Mul), 1) + min(len(cc.Div), 1) + min(len(cc.Min), 1) + min(len(cc.Max), 1)
	if cc.Abs != nil {
		cnt++
	}
	if cc.Sign != nil {
		cnt++
	}
	if cnt != 1 {
		return nil, errors.New("can only have either add, mul, div, min, max, abs or sign")
	}

	o := new(calcPlugin)

	for idx, cc := range cc.Add {
		f, err := cc.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("add[%d]: %w", idx, err)
		}
		o.add = append(o.add, f)
	}

	for idx, cc := range cc.Mul {
		f, err := cc.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("mul[%d]: %w", idx, err)
		}
		o.mul = append(o.mul, f)
	}

	for idx, cc := range cc.Div {
		f, err := cc.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("div[%d]: %w", idx, err)
		}
		o.div = append(o.div, f)
	}

	for idx, cc := range cc.Min {
		f, err := cc.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("min[%d]: %w", idx, err)
		}
		o.min = append(o.min, f)
	}

	for idx, cc := range cc.Max {
		f, err := cc.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("max[%d]: %w", idx, err)
		}
		o.max = append(o.max, f)
	}

	var err error

	o.abs, err = cc.Abs.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}

	o.sign, err = cc.Sign.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	return o, nil
}

var _ IntGetter = (*calcPlugin)(nil)

func (o *calcPlugin) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		f, err := o.floatGetter()
		return int64(f), err
	}, nil
}

var _ StringGetter = (*calcPlugin)(nil)

func (o *calcPlugin) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		f, err := o.floatGetter()
		return fmt.Sprintf("%c", int(f)), err
	}, nil
}

var _ FloatGetter = (*calcPlugin)(nil)

func (o *calcPlugin) FloatGetter() (func() (float64, error), error) {
	return o.floatGetter, nil
}

func (o *calcPlugin) floatGetter() (float64, error) {
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

	case len(o.min) > 0:
		for idx, p := range o.min {
			v, err := p()
			if err != nil {
				return 0, fmt.Errorf("min[%d]: %w", idx, err)
			}
			if idx == 0 {
				res = v
			} else {
				res = math.Min(res, v)
			}
		}

	case len(o.max) > 0:
		for idx, p := range o.max {
			v, err := p()
			if err != nil {
				return 0, fmt.Errorf("max[%d]: %w", idx, err)
			}
			if idx == 0 {
				res = v
			} else {
				res = math.Max(res, v)
			}
		}

	case o.abs != nil:
		v, err := o.abs()
		if err != nil {
			return 0, fmt.Errorf("abs: %w", err)
		}
		res = math.Abs(v)

	default:
		v, err := o.sign()
		if err != nil {
			return 0, fmt.Errorf("sign: %w", err)
		}
		res = math.Copysign(1, v)
	}

	return res, nil
}
