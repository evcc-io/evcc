package provider

import (
	"strconv"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
)

type constProvider struct {
	str string
	set Config
}

func init() {
	registry.Add("const", NewConstFromConfig)
}

// NewConstFromConfig creates const provider
func NewConstFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Value             string
		pipeline.Settings `mapstructure:",squash"`
		Set               Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	pipe, err := pipeline.New(nil, cc.Settings)
	if err != nil {
		return nil, err
	}

	b, err := pipe.Process([]byte(cc.Value))
	if err != nil {
		return nil, err
	}

	o := &constProvider{
		str: string(b),
		set: cc.Set,
	}

	return o, nil
}

var _ StringProvider = (*constProvider)(nil)

func (o *constProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		return o.str, nil
	}, nil
}

var _ IntProvider = (*constProvider)(nil)

func (o *constProvider) IntGetter() (func() (int64, error), error) {
	val, err := strconv.ParseInt(o.str, 10, 64)
	if err != nil && o.str == "" {
		err = nil
	}

	return func() (int64, error) {
		return val, err
	}, err
}

var _ FloatProvider = (*constProvider)(nil)

func (o *constProvider) FloatGetter() (func() (float64, error), error) {
	val, err := strconv.ParseFloat(o.str, 64)
	if err != nil && o.str == "" {
		err = nil
	}

	return func() (float64, error) {
		return val, err
	}, err
}

var _ BoolProvider = (*constProvider)(nil)

func (o *constProvider) BoolGetter() (func() (bool, error), error) {
	val, err := strconv.ParseBool(o.str)
	if err != nil && o.str == "" {
		err = nil
	}

	return func() (bool, error) {
		return val, err
	}, err
}

var _ SetIntProvider = (*constProvider)(nil)

func (o *constProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(param, o.set)
	if err != nil {
		return nil, err
	}

	val, err := strconv.ParseInt(o.str, 10, 64)
	if err != nil && o.str == "" {
		err = nil
	}

	return func(_ int64) error {
		return set(val)
	}, err
}

var _ SetFloatProvider = (*constProvider)(nil)

func (o *constProvider) FloatSetter(param string) (func(float64) error, error) {
	set, err := NewFloatSetterFromConfig(param, o.set)
	if err != nil {
		return nil, err
	}

	val, err := strconv.ParseFloat(o.str, 64)
	if err != nil && o.str == "" {
		err = nil
	}

	return func(_ float64) error {
		return set(val)
	}, err
}

var _ SetBoolProvider = (*constProvider)(nil)

func (o *constProvider) BoolSetter(param string) (func(bool) error, error) {
	set, err := NewBoolSetterFromConfig(param, o.set)
	if err != nil {
		return nil, err
	}

	val, err := strconv.ParseBool(o.str)
	if err != nil && o.str == "" {
		err = nil
	}

	return func(_ bool) error {
		return set(val)
	}, err
}
