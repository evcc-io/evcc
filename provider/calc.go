package provider

import (
	"errors"
	"fmt"

	"github.com/andig/evcc/util"
)

type calcProvider struct {
	add []func() (float64, error)
}

type calcConfig struct {
	Add []Config `validate:"required" ui:"de=Addieren"`
}

func init() {
	registry.Add("calc", "Aggregation", NewCalcFromConfig, calcConfig{})
}

// NewCalcFromConfig creates calc provider
func NewCalcFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc calcConfig

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

// IntGetter fullfills the required IntProvider interface
// TODO replace with Go sum types
func (o *calcProvider) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		return 0, errors.New("calc: int provider not supported")
	}
}

// FloatGetter returns the aggregation of the individual getters
func (o *calcProvider) FloatGetter() func() (float64, error) {
	return func() (float64, error) {
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
}
