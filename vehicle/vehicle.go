package vehicle

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

//go:generate go run ../cmd/tools/decorate.go -f decorateVehicle -b api.Vehicle -t "api.ChargeState,Status,func() (api.ChargeStatus, error)" -t "api.VehicleRange,Range,func() (int64, error)" -t "api.VehicleOdometer,Odometer,func() (float64, error)" -t "api.VehicleClimater,Climater,func() (bool, error)" -t "api.Resurrector,WakeUp,func() (error)"

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
		embed    `mapstructure:",squash"`
		Soc      provider.Config
		Status   *provider.Config
		Range    *provider.Config
		Odometer *provider.Config
		Climater *provider.Config
		Wakeup   *provider.Config
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

	// decorate vehicle with Status
	var status func() (api.ChargeStatus, error)
	if cc.Status != nil {
		statusG, err := provider.NewStringGetterFromConfig(*cc.Status)
		if err != nil {
			return nil, fmt.Errorf("status: %w", err)
		}
		status = func() (api.ChargeStatus, error) {
			s, err := statusG()
			if err != nil {
				return api.StatusNone, err
			}
			return api.ChargeStatusString(s)
		}
	}

	// decorate vehicle with range
	var rng func() (int64, error)
	if cc.Range != nil {
		rangeG, err := provider.NewIntGetterFromConfig(*cc.Range)
		if err != nil {
			return nil, fmt.Errorf("range: %w", err)
		}
		rng = rangeG
	}

	// decorate vehicle with odometer
	var odo func() (float64, error)
	if cc.Odometer != nil {
		odoG, err := provider.NewFloatGetterFromConfig(*cc.Odometer)
		if err != nil {
			return nil, fmt.Errorf("odometer: %w", err)
		}
		odo = odoG
	}

	// decorate vehicle with climater
	var climater func() (bool, error)
	if cc.Climater != nil {
		climateG, err := provider.NewBoolGetterFromConfig(*cc.Climater)
		if err != nil {
			return nil, fmt.Errorf("climater: %w", err)
		}
		climater = climateG
	}

	// decorate vehicle with wakeup
	var wakeup func() error
	if cc.Wakeup != nil {
		wakeupS, err := provider.NewBoolSetterFromConfig("wakeup", *cc.Wakeup)
		if err != nil {
			return nil, fmt.Errorf("wakeup: %w", err)
		}
		wakeup = func() error {
			return wakeupS(true)
		}
	}

	return decorateVehicle(v, status, rng, odo, climater, wakeup), nil
}

// Soc implements the api.Vehicle interface
func (v *Vehicle) Soc() (float64, error) {
	return v.socG()
}
