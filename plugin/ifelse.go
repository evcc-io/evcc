package plugin

import (
	"context"

	"github.com/evcc-io/evcc/util"
)

type ifElsePlugin struct {
	ctx  context.Context
	i, e Config
}

func init() {
	registry.AddCtx("ifelse", NewIfElseFromConfig)
}

// NewIfElseFromConfig creates ifElse provider
func NewIfElseFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		I Config `mapstructure:"if"`
		E Config `mapstructure:"else"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &ifElsePlugin{
		ctx: ctx,
		i:   cc.I,
		e:   cc.E,
	}

	return o, nil
}

var _ IntSetter = (*ifElsePlugin)(nil)

func (o *ifElsePlugin) IntSetter(param string) (func(int64) error, error) {
	ifS, err := o.i.IntSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	elseS, err := o.e.IntSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return func(val int64) error {
		if val > 0 {
			return ifS(val)
		}
		return elseS(val)
	}, nil
}

var _ BoolSetter = (*ifElsePlugin)(nil)

func (o *ifElsePlugin) BoolSetter(param string) (func(bool) error, error) {
	ifS, err := o.i.BoolSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	elseS, err := o.e.BoolSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return func(val bool) error {
		if val {
			return ifS(val)
		}
		return elseS(val)
	}, nil
}
