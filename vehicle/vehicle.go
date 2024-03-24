package vehicle

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

//go:generate go run ../cmd/tools/decorate.go -f decorateVehicle -b api.Vehicle -t "api.SocLimiter,GetLimitSoc,func() (int64, error)" -t "api.ChargeState,Status,func() (api.ChargeStatus, error)" -t "api.VehicleRange,Range,func() (int64, error)" -t "api.VehicleOdometer,Odometer,func() (float64, error)" -t "api.VehicleClimater,Climater,func() (bool, error)" -t "api.CurrentController,MaxCurrent,func(int64) error" -t "api.CurrentGetter,GetMaxCurrent,func() (float64, error)" -t "api.Resurrector,WakeUp,func() error" -t "api.ChargeController,ChargeEnable,func(bool) error"

// Vehicle is an api.Vehicle implementation with configurable getters and setters.
type Vehicle struct {
	*embed
	socG func() (float64, error)
}

func init() {
	registry.Add(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new Vehicle
func NewConfigurableFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	var cc struct {
		embed         `mapstructure:",squash"`
		Soc           provider.Config
		LimitSoc      *provider.Config
		Status        *provider.Config
		Range         *provider.Config
		Odometer      *provider.Config
		Climater      *provider.Config
		MaxCurrent    *provider.Config
		GetMaxCurrent *provider.Config
		Wakeup        *provider.Config
		ChargeEnable  *provider.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	socG, err := provider.NewFloatGetterFromConfig(cc.Soc)
	if err != nil {
		return nil, fmt.Errorf("soc: %w", err)
	}

	v := &Vehicle{
		embed: &cc.embed,
		socG:  socG,
	}

	// decorate range
	var limitSoc func() (int64, error)
	if cc.LimitSoc != nil {
		limitSoc, err = provider.NewIntGetterFromConfig(*cc.LimitSoc)
		if err != nil {
			return nil, fmt.Errorf("limitSoc: %w", err)
		}
	}

	// decorate status
	var status func() (api.ChargeStatus, error)
	if cc.Status != nil {
		get, err := provider.NewStringGetterFromConfig(*cc.Status)
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
	var rng func() (int64, error)
	if cc.Range != nil {
		rng, err = provider.NewIntGetterFromConfig(*cc.Range)
		if err != nil {
			return nil, fmt.Errorf("range: %w", err)
		}
	}

	// decorate odometer
	var odo func() (float64, error)
	if cc.Odometer != nil {
		odo, err = provider.NewFloatGetterFromConfig(*cc.Odometer)
		if err != nil {
			return nil, fmt.Errorf("odometer: %w", err)
		}
	}

	// decorate climater
	var climater func() (bool, error)
	if cc.Climater != nil {
		climater, err = provider.NewBoolGetterFromConfig(*cc.Climater)
		if err != nil {
			return nil, fmt.Errorf("climater: %w", err)
		}
	}

	// decorate maxCurrent
	var maxCurrent func(int64) error
	if cc.MaxCurrent != nil {
		maxCurrent, err = provider.NewIntSetterFromConfig("maxCurrent", *cc.MaxCurrent)
		if err != nil {
			return nil, fmt.Errorf("maxCurrent: %w", err)
		}
	}

	// decorate getMaxCurrent
	var getMaxCurrent func() (float64, error)
	if cc.GetMaxCurrent != nil {
		getMaxCurrent, err = provider.NewFloatGetterFromConfig(*cc.GetMaxCurrent)
		if err != nil {
			return nil, fmt.Errorf("getMaxCurrent: %w", err)
		}
	}

	// decorate wakeup
	var wakeup func() error
	if cc.Wakeup != nil {
		set, err := provider.NewBoolSetterFromConfig("wakeup", *cc.Wakeup)
		if err != nil {
			return nil, fmt.Errorf("wakeup: %w", err)
		}
		wakeup = func() error {
			return set(true)
		}
	}

	// decorate chargeEnable
	var chargeEnable func(bool) error
	if cc.ChargeEnable != nil {
		chargeEnable, err = provider.NewBoolSetterFromConfig("chargeEnable", *cc.ChargeEnable)
		if err != nil {
			return nil, fmt.Errorf("chargeEnable: %w", err)
		}
	}

	return decorateVehicle(v, limitSoc, status, rng, odo, climater, maxCurrent, getMaxCurrent, wakeup, chargeEnable), nil
}

// Soc implements the api.Vehicle interface
func (v *Vehicle) Soc() (float64, error) {
	return v.socG()
}
