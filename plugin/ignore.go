package plugin

import (
	"context"
	"strings"

	"github.com/evcc-io/evcc/util"
)

type ignorePlugin struct {
	ctx context.Context
	err string
	set Config
}

func init() {
	registry.AddCtx("ignore", NewIgnoreFromConfig)
}

// NewIgnoreFromConfig creates const provider
func NewIgnoreFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		Error string
		Set   Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &ignorePlugin{
		ctx: ctx,
		err: cc.Error,
		set: cc.Set,
	}

	return o, nil
}

var _ IntSetter = (*ignorePlugin)(nil)

func ignoreError[T any](fun func(T) error, match string) func(T) error {
	return func(val T) error {
		if err := fun(val); err != nil && !strings.HasPrefix(err.Error(), match) {
			return err
		}
		return nil
	}
}

func (o *ignorePlugin) IntSetter(param string) (func(int64) error, error) {
	set, err := o.set.IntSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.err), nil
}

var _ FloatSetter = (*ignorePlugin)(nil)

func (o *ignorePlugin) FloatSetter(param string) (func(float64) error, error) {
	set, err := o.set.FloatSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.err), nil
}

var _ BoolSetter = (*ignorePlugin)(nil)

func (o *ignorePlugin) BoolSetter(param string) (func(bool) error, error) {
	set, err := o.set.BoolSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.err), nil
}

var _ BytesSetter = (*ignorePlugin)(nil)

func (o *ignorePlugin) BytesSetter(param string) (func([]byte) error, error) {
	set, err := o.set.BytesSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return ignoreError(set, o.err), nil
}
