package plugin

import (
	"context"
	"math"
	"math/rand/v2"

	"github.com/evcc-io/evcc/util"
)

type randomPlugin struct {
	ctx context.Context
	set Config
}

func init() {
	registry.AddCtx("random", NewRandomFromConfig)
}

// NewRandomFromConfig creates random provider
func NewRandomFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		Set Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &randomPlugin{
		ctx: ctx,
		set: cc.Set,
	}

	return o, nil
}

var _ IntSetter = (*randomPlugin)(nil)

func (o *randomPlugin) IntSetter(param string) (func(int64) error, error) {
	set, err := o.set.IntSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return func(int64) error {
		return set(rand.Int64N(math.MaxInt64-1) + 1)
	}, nil
}
