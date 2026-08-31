package plugin

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/evcc-io/evcc/api"
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

		// unmatched value means the device does not implement this setting
		return fmt.Errorf("switch: value not found: %d: %w", val, api.ErrNotAvailable)
	}, nil
}

var _ IntValues = (*switchPlugin)(nil)

// IntValues returns the case values, skipping the cases that only error out.
// A default that sets accepts any other value, too.
func (o *switchPlugin) IntValues() []int64 {
	if o.dflt != nil && sets(o.dflt.intValues(o.ctx)) {
		return nil
	}

	res := make([]int64, 0, len(o.values))
	for i := range o.cases {
		if sets(o.cases[i].Set.intValues(o.ctx)) {
			res = append(res, o.values[i])
		}
	}

	return res
}

// sets is true if the plugin accepts any value at all
func sets(values []int64) bool {
	return values == nil || len(values) > 0
}
