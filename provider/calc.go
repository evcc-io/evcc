package provider

import (
	"github.com/andig/evcc/util"
)

type calcProvider struct {
	add []func() (float64, error)
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(other map[string]interface{}) (func() (float64, error), error) {
	cc := struct {
		Add []Config `validate:"gt=0"`
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &calcProvider{}

	for _, cc := range cc.Add {
		f, err := NewFloatGetterFromConfig(cc)
		if err != nil {
			return nil, err
		}
		o.add = append(o.add, f)
	}

	return o.floatGetter, nil
}

func (o *calcProvider) floatGetter() (float64, error) {
	var sum float64
	for _, p := range o.add {
		v, err := p()
		if err != nil {
			return 0, err
		}
		sum += v
	}

	return sum, nil
}
