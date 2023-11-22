package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type mapProvider struct {
	values map[int64]int64
	set    Config
}

func init() {
	registry.Add("map", NewMapFromConfig)
}

// NewMapFromConfig creates type conversion provider
func NewMapFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Values map[int64]int64
		Set    Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if len(cc.Values) == 0 {
		return nil, fmt.Errorf("missing values")
	}

	o := &mapProvider{
		set:    cc.Set,
		values: cc.Values,
	}

	return o, nil
}

var _ SetIntProvider = (*mapProvider)(nil)

func (o *mapProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(param, o.set)

	return func(val int64) error {
		m, ok := o.values[val]
		if !ok {
			return fmt.Errorf("map: value not found: %d", val)
		}
		return set(m)
	}, err
}
