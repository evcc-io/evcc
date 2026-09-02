package curtailer

import (
	"context"
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a curtailer from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Curtailer, error) {
	cc := struct {
		measurement.Curtailer `mapstructure:",squash"`
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	curtailS, curtailedG, err := cc.Curtailer.Configure(ctx)
	if err != nil {
		return nil, err
	}

	if curtailS == nil {
		return nil, errors.New("missing curtail")
	}

	return implement.Curtailer(
		func() (int, error) {
			percent, err := curtailedG()
			return int(percent), err
		},
		func(percent int) error {
			return curtailS(int64(percent))
		},
	), nil
}
