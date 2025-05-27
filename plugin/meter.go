package plugin

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	meter "github.com/evcc-io/evcc/meter/config"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type meterPlugin struct {
	meter api.Meter
	key   Key
}

func init() {
	registry.AddCtx("meter", NewMeterFromConfig)
}

// NewMeterFromConfig creates type conversion provider
func NewMeterFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		Config config.Typed
		Key
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	meter, err := meter.NewFromConfig(ctx, cc.Config.Type, cc.Config.Other)
	if err != nil {
		return nil, err
	}

	o := &meterPlugin{
		meter: meter,
		key:   cc.Key,
	}

	return o, nil
}

var _ FloatGetter = (*meterPlugin)(nil)

func (o *meterPlugin) FloatGetter() (func() (float64, error), error) {
	err := fmt.Errorf("unsupported reading %s", o.key)

	switch o.key {
	case Power:
	case Energy:
		if _, ok := o.meter.(api.MeterEnergy); !ok {
			return nil, err
		}
	default:
		return nil, err
	}

	return func() (float64, error) {
		switch o.key {
		case Power:
			return o.meter.CurrentPower()
		case Energy:
			return o.meter.(api.MeterEnergy).TotalEnergy()
		default:
			return 0, err
		}
	}, nil
}
