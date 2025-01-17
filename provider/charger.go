package provider

import (
	"context"

	"github.com/evcc-io/evcc/api"
	charger "github.com/evcc-io/evcc/charger/config"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cast"
)

type switchChargerProvider struct {
	charger api.Charger
}

func init() {
	registry.AddCtx("charger", NewChargerEnableFromConfig)
}

// NewChargerEnableFromConfig creates type conversion provider
func NewChargerEnableFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Config config.Typed
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	charger, err := charger.NewFromConfig(ctx, cc.Config.Type, cc.Config.Other)
	if err != nil {
		return nil, err
	}

	o := &switchChargerProvider{
		charger: charger,
	}

	return o, nil
}

var _ FloatProvider = (*switchChargerProvider)(nil)

func (o *switchChargerProvider) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		v, err := o.charger.Enabled()
		return cast.ToFloat64(v), err
	}, nil
}

var _ IntProvider = (*switchChargerProvider)(nil)

func (o *switchChargerProvider) IntGetter() (func() (int64, error), error) {
	return func() (int64, error) {
		v, err := o.charger.Enabled()
		return cast.ToInt64(v), err
	}, nil
}

var _ SetIntProvider = (*switchChargerProvider)(nil)

func (o *switchChargerProvider) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		b, err := cast.ToBoolE(val)
		if err != nil {
			return err
		}
		return o.charger.Enable(b)
	}, nil
}

var _ BoolProvider = (*switchChargerProvider)(nil)

func (o *switchChargerProvider) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		return o.charger.Enabled()
	}, nil
}

var _ SetBoolProvider = (*switchChargerProvider)(nil)

func (o *switchChargerProvider) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return o.charger.Enable(val)
	}, nil
}
