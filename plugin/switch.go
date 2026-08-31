package plugin

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/evcc-io/evcc/util"
)

type Case struct {
	Case string
	Set  Config
}

type switchPlugin struct {
	ctx    context.Context
	cases  []Case
	values []int64
	dflt   *Config
}

func init() {
	registry.AddCtx("switch", NewSwitchFromConfig)
}

// NewSwitchFromConfig creates switch provider
func NewSwitchFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Switch  []Case
		Default *Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	values := make([]int64, 0, len(cc.Switch))
	for _, c := range cc.Switch {
		val, err := strconv.ParseInt(c.Case, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("switch: invalid case: %s", c.Case)
		}
		if slices.Contains(values, val) {
			return nil, fmt.Errorf("switch: duplicate case: %s", c.Case)
		}
		values = append(values, val)
	}

	o := &switchPlugin{
		ctx:    ctx,
		cases:  cc.Switch,
		values: values,
		dflt:   cc.Default,
	}

	return o, nil
}

var _ IntSetter = (*switchPlugin)(nil)

func (o *switchPlugin) IntSetter(param string) (func(int64) error, error) {
	set := make([]func(int64) error, 0, len(o.cases))
	for _, cc := range o.cases {
		s, err := cc.Set.IntSetter(o.ctx, param)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	dflt, err := o.dflt.IntSetter(o.ctx, param)
	if err != nil {
		return nil, err
	}

	return func(val int64) error {
		if i := slices.Index(o.values, val); i >= 0 {
			return set[i](val)
		}

		if dflt != nil {
			return dflt(val)
		}

		return fmt.Errorf("switch: value not found: %d", val)
	}, nil
}

var _ IntKeysGetter = (*switchPlugin)(nil)

// IntKeys returns the values the switch has a case for. A default accepts the
// remaining values too, so the switch then has no fixed set of keys.
func (o *switchPlugin) IntKeys() ([]int64, error) {
	if o.dflt != nil {
		return nil, nil
	}

	return slices.Clone(o.values), nil
}
