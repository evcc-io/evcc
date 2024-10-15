package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/util"
)

type Case struct {
	Case string
	Set  Config
}

type switchProvider struct {
	ctx   context.Context
	cases []Case
	dflt  *Config
}

func init() {
	registry.AddCtx("switch", NewSwitchFromConfig)
}

// NewSwitchFromConfig creates switch provider
func NewSwitchFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Switch  []Case
		Default *Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	cases := make(map[string]struct{})
	for _, c := range cc.Switch {
		if _, ok := cases[c.Case]; ok {
			return nil, fmt.Errorf("switch: duplicate case: %s", c.Case)
		}
		cases[c.Case] = struct{}{}
	}

	o := &switchProvider{
		ctx:   ctx,
		cases: cc.Switch,
		dflt:  cc.Default,
	}

	return o, nil
}

var _ SetIntProvider = (*switchProvider)(nil)

func (o *switchProvider) IntSetter(param string) (func(int64) error, error) {
	set := make([]func(int64) error, 0, len(o.cases))
	for _, cc := range o.cases {
		s, err := NewIntSetterFromConfig(o.ctx, param, cc.Set)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	var dflt func(int64) error
	if o.dflt != nil {
		var err error
		if dflt, err = NewIntSetterFromConfig(o.ctx, param, *o.dflt); err != nil {
			return nil, err
		}
	}

	return func(val int64) error {
		for i, s := range o.cases {
			ival, err := strconv.ParseInt(s.Case, 10, 64)
			if err != nil {
				return err
			}

			if ival == val {
				return set[i](val)
			}
		}

		if dflt != nil {
			return dflt(val)
		}

		return fmt.Errorf("switch: value not found: %d", val)
	}, nil
}
