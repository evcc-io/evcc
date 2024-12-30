package provider

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type enableProvider struct {
	charger api.Charger
}

func init() {
	registry.AddCtx("map", NewMapFromConfig)
}

// NewEnableFromConfig creates type conversion provider
func NewEnableFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Charger config.Typed
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Charger.Type == "" {
		return nil, fmt.Errorf("missing charger")
	}

	charger, err := charger.NewFromConfig(ctx, cc.Charger.Type, cc.Charger.Other)
	if err != nil {
		return nil, err
	}

	o := &enableProvider{
		charger: charger,
	}

	return o, nil
}

var _ BoolProvider = (*enableProvider)(nil)

func (o *enableProvider) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		return o.charger.Enabled()
	}, nil
}

var _ SetBoolProvider = (*enableProvider)(nil)

func (o *enableProvider) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return o.charger.Enable(val)
	}, nil
}
