package plugin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
)

// fallbackPlugin provides a configured value, or reads it once from the fallback source if unset.
// As setter it writes that value to the set target, ignoring the value passed in.
type fallbackPlugin struct {
	ctx   context.Context
	scale float64
	set   *Config

	mu     sync.Mutex
	val    float64
	read   bool
	source func() (float64, error)
}

func init() {
	registry.AddCtx("fallback", NewFallbackFromConfig)
}

// NewFallbackFromConfig creates fallback provider
func NewFallbackFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Value    string
		Fallback *Config
		Scale    float64
		Set      *Config
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if !isFinite(cc.Scale) {
		return nil, fmt.Errorf("invalid scale: %v", cc.Scale)
	}

	p := &fallbackPlugin{
		ctx:   ctx,
		scale: cc.Scale,
		set:   cc.Set,
	}

	if cc.Value != "" {
		val, err := strconv.ParseFloat(cc.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("value: %w", err)
		}
		if !isFinite(val) {
			return nil, fmt.Errorf("invalid value: %v", val)
		}
		if val != 0 {
			p.val, p.read = val, true
			return p, nil
		}
	}

	if cc.Fallback == nil {
		return nil, errors.New("missing value or fallback")
	}

	source, err := numericGetter(ctx, cc.Fallback)
	if err != nil {
		return nil, fmt.Errorf("fallback: %w", err)
	}
	p.source = source

	return p, nil
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// numericGetter creates a float getter from a source providing float or int values
func numericGetter(ctx context.Context, c *Config) (func() (float64, error), error) {
	if c.Source == "" {
		return nil, errors.New("missing plugin source")
	}

	factory, err := registry.Get(c.Source)
	if err != nil {
		return nil, err
	}

	prov, err := factory(ctx, c.Other)
	if err != nil {
		return nil, err
	}

	fg, isFloat := prov.(FloatGetter)
	ig, isInt := prov.(IntGetter)

	var getFloat func() (float64, error)
	if isFloat {
		if getFloat, err = fg.FloatGetter(); err != nil {
			return nil, err
		}
	}

	var getInt func() (int64, error)
	if isInt {
		if getInt, err = ig.IntGetter(); err != nil {
			return nil, err
		}
	}

	if getFloat == nil && getInt == nil {
		return nil, fmt.Errorf("invalid plugin source for type float or int: %s", c.Source)
	}

	// sources like go only succeed through the getter matching their result type
	return func() (float64, error) {
		var ferr error
		if getFloat != nil {
			v, err := getFloat()
			if err == nil || getInt == nil {
				return v, err
			}
			ferr = err
		}

		v, err := getInt()
		if err != nil {
			return 0, errors.Join(ferr, err)
		}
		return float64(v), nil
	}, nil
}

// value returns the configured value or the first valid, non-zero fallback reading
func (p *fallbackPlugin) value() (float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.read {
		v, err := p.source()
		if err != nil {
			return 0, fmt.Errorf("fallback: %w", err)
		}
		if !isFinite(v) {
			return 0, fmt.Errorf("fallback: invalid value: %v", v)
		}
		// zero means unavailable, e.g. a decoded NaN register
		if v == 0 {
			return 0, errors.New("fallback: no value")
		}
		p.val, p.read = v, true
	}

	v := p.scale * p.val
	if !isFinite(v) {
		return 0, fmt.Errorf("invalid value: %v", v)
	}

	return v, nil
}

// intValue returns the rounded value, rejecting values outside the int64 range
func (p *fallbackPlugin) intValue() (int64, error) {
	v, err := p.value()
	if err != nil {
		return 0, err
	}

	v = math.Round(v)
	if v < math.MinInt64 || v >= math.MaxInt64 {
		return 0, fmt.Errorf("value out of range: %v", v)
	}

	return int64(v), nil
}

var _ FloatGetter = (*fallbackPlugin)(nil)

func (p *fallbackPlugin) FloatGetter() (func() (float64, error), error) {
	return p.value, nil
}

var _ IntGetter = (*fallbackPlugin)(nil)

func (p *fallbackPlugin) IntGetter() (func() (int64, error), error) {
	return p.intValue, nil
}

var _ FloatSetter = (*fallbackPlugin)(nil)

func (p *fallbackPlugin) FloatSetter(param string) (func(float64) error, error) {
	if p.set == nil {
		return nil, errors.New("missing set")
	}

	set, err := p.set.FloatSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	return func(_ float64) error {
		v, err := p.value()
		if err != nil {
			return err
		}
		return set(v)
	}, nil
}

var _ IntSetter = (*fallbackPlugin)(nil)

func (p *fallbackPlugin) IntSetter(param string) (func(int64) error, error) {
	if p.set == nil {
		return nil, errors.New("missing set")
	}

	set, err := p.set.IntSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	return func(_ int64) error {
		v, err := p.intValue()
		if err != nil {
			return err
		}
		return set(v)
	}, nil
}
