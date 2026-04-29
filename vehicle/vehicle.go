package vehicle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

//go:generate go tool decorate -f decorateVehicle -b api.Vehicle -t api.SocLimiter,api.ChargeState,api.VehicleRange,api.VehicleOdometer,api.VehicleClimater,api.CurrentController,api.CurrentGetter,api.VehicleFinishTimer,api.Resurrector,api.ChargeController,api.ChargeRater,api.VehiclePosition

// Vehicle is an api.Vehicle implementation with configurable getters and setters.
type Vehicle struct {
	*embed
	socG func() (float64, error)
}

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new vehicle from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	var cc struct {
		embed         `mapstructure:",squash"`
		Soc           plugin.Config
		LimitSoc      *plugin.Config
		Status        *plugin.Config
		Range         *plugin.Config
		Odometer      *plugin.Config
		Climater      *plugin.Config
		MaxCurrent    *plugin.Config
		GetMaxCurrent *plugin.Config
		FinishTime    *plugin.Config
		Wakeup        *plugin.Config
		ChargeEnable  *plugin.Config
		ChargedEnergy *plugin.Config
		Position      *struct {
			Latitude  plugin.Config `mapstructure:"latitude"`
			Longitude plugin.Config `mapstructure:"longitude"`
		} `mapstructure:"position"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	socG, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("soc: %w", err)
	}

	v := &Vehicle{
		embed: &cc.embed,
		socG:  socG,
	}

	// decorate range
	limitSoc, err := cc.LimitSoc.IntGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("limitSoc: %w", err)
	}

	// decorate status
	var status func() (api.ChargeStatus, error)
	if cc.Status != nil {
		get, err := cc.Status.StringGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("status: %w", err)
		}
		status = func() (api.ChargeStatus, error) {
			s, err := get()
			if err != nil {
				return api.StatusNone, err
			}
			return api.ChargeStatusString(s)
		}
	}

	// decorate range
	rng, err := cc.Range.IntGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("range: %w", err)
	}

	// decorate odometer
	odo, err := cc.Odometer.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("odometer: %w", err)
	}

	// decorate climater
	climater, err := cc.Climater.BoolGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("climater: %w", err)
	}

	// decorate maxCurrent
	maxCurrent, err := cc.MaxCurrent.IntSetter(ctx, "maxcurrent")
	if err != nil {
		return nil, fmt.Errorf("maxCurrent: %w", err)
	}

	// decorate getMaxCurrent
	getMaxCurrent, err := cc.GetMaxCurrent.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("getMaxCurrent: %w", err)
	}

	// decorate position
	var position func() (float64, float64, error)
	if cc.Position != nil {
		latG, err := cc.Position.Latitude.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("latitude: %w", err)
		}
		lonG, err := cc.Position.Longitude.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("longitude: %w", err)
		}
		position = func() (float64, float64, error) {
			lat, err := latG()
			if err != nil {
				return 0, 0, err
			}
			lon, err := lonG()
			if err != nil {
				return 0, 0, err
			}
			// MQTT sources may report (0,0) when no GPS fix is available
			if lat == 0 && lon == 0 {
				return 0, 0, api.ErrNotAvailable
			}
			return lat, lon, nil
		}
	}

	// decorate finishtime
	var finishTime func() (time.Time, error)
	if cc.FinishTime != nil {
		stringG, err := cc.FinishTime.StringGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("finishTime: %w", err)
		}
		finishTime = func() (time.Time, error) {
			s, err := stringG()
			if err != nil {
				return time.Time{}, err
			}
			return time.Parse(time.RFC3339, s)
		}
	}

	// decorate wakeup
	var wakeup func() error
	if cc.Wakeup != nil {
		set, err := cc.Wakeup.BoolSetter(ctx, "wakeup")
		if err != nil {
			return nil, fmt.Errorf("wakeup: %w", err)
		}
		wakeup = func() error {
			return set(true)
		}
	}

	// decorate chargeEnable
	chargeEnable, err := cc.ChargeEnable.BoolSetter(ctx, "chargeenable")
	if err != nil {
		return nil, fmt.Errorf("chargeEnable: %w", err)
	}

	// decorate chargedenergy
	var chargedEnergy func() (float64, error)
	if cc.ChargedEnergy != nil {
		var err error
		chargedEnergy, err = cc.ChargedEnergy.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("charged energy: %w", err)
		}
	}

	switch {
	case maxCurrent == nil && getMaxCurrent != nil:
		return nil, errors.New("cannot have current without current control")
	case status == nil && maxCurrent != nil:
		return nil, errors.New("cannot have current control without status")
	case status == nil && chargeEnable != nil:
		return nil, errors.New("cannot have charge control without status")
	}

	return decorateVehicle(v, limitSoc, status, rng, odo, climater, maxCurrent, getMaxCurrent, finishTime, wakeup, chargeEnable, chargedEnergy, position), nil
}

// Soc implements the api.Vehicle interface
func (v *Vehicle) Soc() (float64, error) {
	return v.socG()
}
