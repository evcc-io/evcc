package provider

import (
	"github.com/andig/evcc/util"
)

type calcProvider struct {
	add []FloatGetter
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(log *util.Logger, other map[string]interface{}) FloatGetter {
	cc := struct {
		Add []Config
	}{}
	util.DecodeOther(log, other, &cc)

	o := &calcProvider{}

	for _, cc := range cc.Add {
		o.add = append(o.add, NewFloatGetterFromConfig(log, cc))
	}

	return o.floatGetter
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
