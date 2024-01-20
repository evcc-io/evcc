package provider

import (
	"strconv"

	"github.com/evcc-io/evcc/util"
)

type fallbackProvider struct {
	log     *util.Logger
	str     string
	initial bool
	get     Config
}

func init() {
	registry.Add("fallback", NewFallbackFromConfig)
}

// NewFallbackFromConfig creates fallback provider
func NewFallbackFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Value   string
		Initial bool
		Get     Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &fallbackProvider{
		log:     util.NewLogger("fallback"),
		str:     cc.Value,
		initial: cc.Initial,
		get:     cc.Get,
	}

	return o, nil
}

var _ IntProvider = (*fallbackProvider)(nil)

func (o *fallbackProvider) IntGetter() (func() (int64, error), error) {
	val, err := strconv.ParseInt(o.str, 10, 64)
	if err != nil {
		return nil, err
	}

	g, err := NewIntGetterFromConfig(o.get)

	// initial fallback, e.g. for sunspec model not existing
	if err != nil && o.initial {
		return func() (int64, error) {
			return val, nil
		}, nil
	}

	return func() (int64, error) {
		v, err := g()
		if err != nil {
			o.log.DEBUG.Printf("fallback: %v, cause %v", val, err)
			return val, nil
		}

		return v, nil
	}, err
}

var _ FloatProvider = (*fallbackProvider)(nil)

func (o *fallbackProvider) FloatGetter() (func() (float64, error), error) {
	val, err := strconv.ParseFloat(o.str, 64)
	if err != nil {
		return nil, err
	}

	g, err := NewFloatGetterFromConfig(o.get)

	// initial fallback, e.g. for sunspec model not existing
	if err != nil && o.initial {
		return func() (float64, error) {
			return val, nil
		}, nil
	}

	return func() (float64, error) {
		v, err := g()
		if err != nil {
			o.log.DEBUG.Printf("fallback: %v, cause %v", val, err)
			return val, nil
		}

		return v, nil
	}, err
}
