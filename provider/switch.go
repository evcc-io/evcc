package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type Case struct {
	Case int64
	Set  Config
}

type switchProvider struct {
	set  []Case
	dflt *Config
}

func init() {
	registry.Add("switch", NewSwitchFromConfig)
}

// NewSwitchFromConfig creates switch provider
func NewSwitchFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Set     []Case
		Default *Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &switchProvider{
		set:  cc.Set,
		dflt: cc.Default,
	}

	return o, nil
}

var _ SetIntProvider = (*switchProvider)(nil)

func (o *switchProvider) IntSetter(param string) func(int64) error {
	set := make([]func(int64) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := NewIntSetterFromConfig(param, cc.Set)
		if err != nil {
			_ = err
		}
		set = append(set, s)
	}

	var dflt func(int64) error
	if o.dflt != nil {
		var err error
		if dflt, err = NewIntSetterFromConfig(param, *o.dflt); err != nil {
			_ = err
		}
	}

	return func(val int64) error {
		for i, s := range o.set {
			if s.Case == val {
				return set[i](val)
			}
		}

		if dflt != nil {
			return dflt(val)
		}

		return fmt.Errorf("value %d not found", val)
	}
}
