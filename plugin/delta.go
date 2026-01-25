package plugin

import (
	"context"

	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
)

type deltaPlugin struct {
	ctx   context.Context
	total float64
	set   Config
	get   *Config
}

func init() {
	registry.AddCtx("delta", NewDeltaFromConfig)
}

// NewDeltaFromConfig creates delta provider
func NewDeltaFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		pipeline.Settings `mapstructure:",squash"`
		Set               Config
		Get               *Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := &deltaPlugin{
		ctx: ctx,
		set: cc.Set,
		get: cc.Get,
	}

	return p, nil
}

var _ IntSetter = (*deltaPlugin)(nil)

func (p *deltaPlugin) IntSetter(param string) (func(int64) error, error) {
	set, err := p.set.IntSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	get, err := p.get.IntGetter(p.ctx)
	if err != nil {
		return nil, err
	}

	return func(val int64) error {
		if get != nil {
			total, err := get()
			if err != nil {
				return err
			}
			p.total = float64(total)
		}

		delta := float64(val) - p.total
		err := set(int64(delta))
		if err == nil {
			p.total = float64(val)
		}
		return err
	}, err
}

var _ FloatSetter = (*deltaPlugin)(nil)

func (p *deltaPlugin) FloatSetter(param string) (func(float64) error, error) {
	set, err := p.set.FloatSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	get, err := p.get.FloatGetter(p.ctx)
	if err != nil {
		return nil, err
	}

	return func(val float64) error {
		if get != nil {
			total, err := get()
			if err != nil {
				return err
			}
			p.total = total
		}

		delta := val - p.total
		err := set(delta)
		if err == nil {
			p.total = val
		}
		return err
	}, err
}
