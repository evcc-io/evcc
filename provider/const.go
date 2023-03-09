package provider

import (
	"strconv"

	"github.com/evcc-io/evcc/util"
)

type constProvider struct {
	str string
}

func init() {
	registry.Add("const", NewConstFromConfig)
}

// NewConstFromConfig creates const provider
func NewConstFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc struct {
		Value string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &constProvider{
		str: cc.Value,
	}

	return o, nil
}

func (o *constProvider) StringGetter() func() (string, error) {
	return func() (string, error) {
		return o.str, nil
	}
}

func (o *constProvider) IntGetter() func() (int64, error) {
	val, err := strconv.ParseInt(o.str, 10, 64)
	return func() (int64, error) {
		return val, err
	}
}

func (o *constProvider) FloatGetter() func() (float64, error) {
	val, err := strconv.ParseFloat(o.str, 64)
	return func() (float64, error) {
		return val, err
	}
}
