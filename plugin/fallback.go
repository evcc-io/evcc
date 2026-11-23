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
		if val != 0 {
			p.val, p.read = val, true
			return p, nil
		}
	}

	if cc.Fallback == nil {
		return nil, errors.New("missing value or fallback")
	}

	source, err := cc.Fallback.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallback: %w", err)
	}
	p.source = source

	return p, nil
}

// value returns the configured value or the first successful fallback reading
func (p *fallbackPlugin) value() (float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.read {
		v, err := p.source()
		if err != nil {
			return 0, fmt.Errorf("fallback: %w", err)
		}
		p.val, p.read = v, true
	}

	return p.scale * p.val, nil
}

var _ FloatGetter = (*fallbackPlugin)(nil)

func (p *fallbackPlugin) FloatGetter() (func() (float64, error), error) {
	return p.value, nil
}

var _ IntGetter = (*fallbackPlugin)(nil)

func (p *fallbackPlugin) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		v, err := p.value()
		return int64(math.Round(v)), err
	}, nil
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
		v, err := p.value()
		if err != nil {
			return err
		}
		return set(int64(math.Round(v)))
	}, nil
}
