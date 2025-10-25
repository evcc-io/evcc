package charger

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/measurement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

func init() {
	registry.AddCtx("sgready-relay", NewSgReadyRelayFromConfig)
	registry.AddCtx("sgready-boost", NewSgReadyRelayFromConfig) // TODO deprecated
}

// NewSgReadyRelayFromConfig creates an SG Ready charger with boost/dim relays from generic config
func NewSgReadyRelayFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed                   `mapstructure:",squash"`
		Boost                   config.Typed
		Dim                     *config.Typed
		measurement.Temperature `mapstructure:",squash"`
		measurement.Energy      `mapstructure:",squash"`
	}{
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	boost, err := NewFromConfig(ctx, cc.Boost.Type, cc.Boost.Other)
	if err != nil {
		return nil, err
	}

	dim, err := NewFromConfig(ctx, cc.Dim.Type, cc.Dim.Other)
	if err != nil {
		return nil, err
	}

	tempG, limitTempG, err := cc.Temperature.Configure(ctx)
	if err != nil {
		return nil, err
	}

	res, err := NewSgReadyRelay(ctx, &cc.embed, boost, dim)
	if err != nil {
		return nil, err
	}

	powerG, energyG, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}

	return decorateSgReady(res, powerG, energyG, tempG, limitTempG), nil
}

// NewSgReadyRelay creates SG Ready charger with boost relay
func NewSgReadyRelay(ctx context.Context, embed *embed, boost, dim api.Charger) (*SgReady, error) {
	modeS := func(mode int64) error {
		switch mode {
		case Dim:
			if dim == nil {
				return api.ErrNotAvailable
			}
			if err := boost.Enable(false); err != nil {
				return err
			}
			return dim.Enable(true)

		case Normal:
			if dim != nil {
				if err := dim.Enable(false); err != nil {
					return err
				}
			}
			return boost.Enable(false)

		case Boost:
			if dim != nil {
				if err := dim.Enable(false); err != nil {
					return err
				}
			}
			return boost.Enable(true)

		default:
			return fmt.Errorf("invalid sgready mode: %d", mode)
		}
	}

	modeG := func() (int64, error) {
		if dim != nil {
			dimmed, err := dim.Enabled()
			if err != nil {
				return 0, err
			}
			if dimmed {
				return Dim, nil
			}
		}

		boosted, err := boost.Enabled()
		if err != nil {
			return 0, err
		}

		if boosted {
			return Boost,nil
		}

		return Normal, nil
	}

	return NewSgReady(ctx, embed, modeS, modeG, nil)
}
