package provider

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type mapProvider struct {
	ctx      context.Context
	values   map[int64]int64
	get, set Config
}

func init() {
	registry.AddCtx("map", NewMapFromConfig)
}

// NewMapFromConfig creates type conversion provider
func NewMapFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
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

	o := &mapProvider{
		ctx:    ctx,
		get:    cc.Get,
		set:    cc.Set,
		values: cc.Values,
	}

	return o, nil
}

var _ IntProvider = (*mapProvider)(nil)

func (o *mapProvider) IntGetter() (func() (int64, error), error) {
	get, err := NewIntGetterFromConfig(o.ctx, o.get)

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

var _ SetIntProvider = (*mapProvider)(nil)

func (o *mapProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(o.ctx, param, o.set)

	return func(val int64) error {
		m, ok := o.values[val]
		if !ok {
			return fmt.Errorf("map: value not found: %d", val)
		}
		return set(m)
	}, err
}
