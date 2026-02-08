package plugin

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type mapPlugin struct {
	ctx      context.Context
	values   map[int64]int64
	get, set Config
}

func init() {
	registry.AddCtx("map", NewMapFromConfig)
}

// NewMapFromConfig creates type conversion provider
func NewMapFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Values   map[int64]int64
		Get, Set Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if len(cc.Values) == 0 {
		return nil, fmt.Errorf("missing values")
	}

	o := &mapPlugin{
		ctx:    ctx,
		get:    cc.Get,
		set:    cc.Set,
		values: cc.Values,
	}

	return o, nil
}

var _ IntGetter = (*mapPlugin)(nil)

func (o *mapPlugin) IntGetter() (func() (int64, error), error) {
	get, err := o.get.IntGetter(o.ctx)

	return func() (int64, error) {
		val, err := get()
		if err != nil {
			return 0, err
		}

		res, ok := o.values[val]
		if !ok {
			return 0, fmt.Errorf("map: value not found: %d", val)
		}

		return res, nil
	}, err
}

var _ IntSetter = (*mapPlugin)(nil)

func (o *mapPlugin) IntSetter(param string) (func(int64) error, error) {
	set, err := o.set.IntSetter(o.ctx, param)

	return func(val int64) error {
		m, ok := o.values[val]
		if !ok {
			return fmt.Errorf("map: value not found: %d", val)
		}
		return set(m)
	}, err
}
