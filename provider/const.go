package provider

import (
	"strconv"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
)

type constProvider struct {
	str string
}

func init() {
	registry.Add("const", NewConstFromConfig)
}

// NewConstFromConfig creates const provider
func NewConstFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Value             string
		pipeline.Settings `mapstructure:",squash"`
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
	return func() (int64, error) {
		return val, err
	}, err
}

var _ FloatProvider = (*constProvider)(nil)

func (o *constProvider) FloatGetter() (func() (float64, error), error) {
	val, err := strconv.ParseFloat(o.str, 64)
	return func() (float64, error) {
		return val, err
	}, err
}
