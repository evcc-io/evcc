package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

//go:generate decorate -f decorateVehicle -b api.Vehicle -t "api.SocLimiter,GetLimitSoc,func() (int64, error)" -t "api.ChargeState,Status,func() (api.ChargeStatus, error)" -t "api.VehicleRange,Range,func() (int64, error)" -t "api.VehicleOdometer,Odometer,func() (float64, error)" -t "api.VehicleClimater,Climater,func() (bool, error)" -t "api.CurrentController,MaxCurrent,func(int64) error" -t "api.CurrentGetter,GetMaxCurrent,func() (float64, error)" -t "api.VehicleFinishTimer,FinishTime,func() (time.Time, error)" -t "api.Resurrector,WakeUp,func() error" -t "api.ChargeController,ChargeEnable,func(bool) error"

// Vehicle is an api.Vehicle implementation with configurable getters and setters.
type Vehicle struct {
	*embed
	socG func() (float64, error)
}

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new Vehicle
func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
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

	return decorateVehicle(v, limitSoc, status, rng, odo, climater, maxCurrent, getMaxCurrent, finishTime, wakeup, chargeEnable), nil
}

// Soc implements the api.Vehicle interface
func (v *Vehicle) Soc() (float64, error) {
	return v.socG()
}
