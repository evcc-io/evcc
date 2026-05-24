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
	meter  api.Meter
	method Method
	scale  float64
}

func init() {
	registry.AddCtx("meter", NewMeterFromConfig)
}

// NewMeterFromConfig creates type conversion provider
func NewMeterFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Config config.Typed
		Method
		Scale float64
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	meter, err := meter.NewFromConfig(ctx, cc.Config.Type, cc.Config.Other)
	if err != nil {
		return nil, err
	}

	o := &meterPlugin{
		meter:  meter,
		method: cc.Method,
		scale:  cc.Scale,
	}

	return o, nil
}

var _ FloatGetter = (*meterPlugin)(nil)

func (o *meterPlugin) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		switch o.method {
		case Energy:
			if m, ok := api.Cap[api.MeterEnergy](o.meter); ok {
				f, err := m.TotalEnergy()
				return f * o.scale, err
			}
		case ReturnEnergy:
			if m, ok := api.Cap[api.MeterReturnEnergy](o.meter); ok {
				f, err := m.ReturnEnergy()
				return f * o.scale, err
			}
		case Soc:
			if m, ok := api.Cap[api.Battery](o.meter); ok {
				f, err := m.Soc()
				return f * o.scale, err
			}
		case Power:
			f, err := o.meter.CurrentPower()
			return f * o.scale, err
		}

		return 0, fmt.Errorf("unsupported method: %s", o.method.String())
	}, nil
}
