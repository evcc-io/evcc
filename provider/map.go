package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type mapProvider struct {
	values map[int64]int64
	set    func(int64) error
}

func init() {
	registry.Add("map", NewMapFromConfig)
}

// NewMapFromConfig creates type conversion provider
func NewMapFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Param  string
		Values map[int64]int64
		Set    Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if len(cc.Values) == 0 {
		return nil, fmt.Errorf("missing values")
	}

	// TODO late init
	set, err := NewIntSetterFromConfig(cc.Param, cc.Set)
	if err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}

	o := &mapProvider{
		set:    set,
		values: cc.Values,
	}

	return o, nil
}

var _ SetIntProvider = (*mapProvider)(nil)

func (o *mapProvider) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		m, ok := o.values[val]
		if !ok {
			return fmt.Errorf("value %d not found", val)
		}
		return o.set(m)
	}, nil
}
