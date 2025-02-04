package plugin

import (
	"strconv"

	"github.com/spf13/cast"
)

type getter struct {
	get   StringGetter
	scale float64
}

func defaultGetters(get StringGetter, scale float64) *getter {
	return &getter{
		get:   get,
		scale: scale,
	}
}

var _ FloatGetter = (*getter)(nil)

// FloatGetter parses float from exec result
func (p *getter) FloatGetter() (func() (float64, error), error) {
	g, err := p.get.StringGetter()

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}

		return f * p.scale, nil
	}, err
}

var _ IntGetter = (*getter)(nil)

// IntGetter parses int64 from exec result
func (p *getter) IntGetter() (func() (int64, error), error) {
	g, err := p.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		if err != nil {
			return 0, err
		}

		return int64(f), nil
	}, err
}

var _ BoolGetter = (*getter)(nil)

// BoolGetter parses bool from exec result. "on", "true" and 1 are considered truish.
func (p *getter) BoolGetter() (func() (bool, error), error) {
	g, err := p.get.StringGetter()

	return func() (bool, error) {
		s, err := g()
		if err != nil {
			return false, err
		}

		return cast.ToBoolE(s)
	}, err
}
