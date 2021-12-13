package vehicle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle       *tesla.Vehicle
	chargeStateG  func() (interface{}, error)
	vehicleStateG func() (interface{}, error)
	driveStateG   func() (interface{}, error)
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

// NewTeslaFromConfig creates a new vehicle
func NewTeslaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		Tokens Tokens
		VIN    string
		Cache  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.Tokens.Error(); err != nil {
		return nil, err
	}

	v := &Tesla{
		embed: &cc.embed,
	}

	// authenticated http client with logging injected to the Tesla client
	log := logx.Redact(logx.NewModule("tesla"), cc.Tokens.Access, cc.Tokens.Refresh)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)

	options := []tesla.ClientOption{tesla.WithToken(&oauth2.Token{
		AccessToken:  cc.Tokens.Access,
		RefreshToken: cc.Tokens.Refresh,
		Expiry:       time.Now(),
	})}

	client, err := tesla.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	vehicles, err := client.Vehicles()
	if err != nil {
		return nil, err
	}

	if cc.VIN == "" && len(vehicles) == 1 {
		v.vehicle = vehicles[0]
	} else {
		for _, vehicle := range vehicles {
			if vehicle.Vin == strings.ToUpper(cc.VIN) {
				v.vehicle = vehicle
			}
		}
	}

	if v.vehicle == nil {
		return nil, errors.New("vin not found")
	}

	if v.Title_ == "" {
		v.Title_ = v.vehicle.DisplayName
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).InterfaceGetter()
	v.vehicleStateG = provider.NewCached(v.vehicleState, cc.Cache).InterfaceGetter()
	v.driveStateG = provider.NewCached(v.driveState, cc.Cache).InterfaceGetter()

	return v, nil
}

// chargeState implements the charge state api
func (v *Tesla) chargeState() (interface{}, error) {
	return v.vehicle.ChargeState()
}

// vehicleState implements the climater api
func (v *Tesla) vehicleState() (interface{}, error) {
	return v.vehicle.VehicleState()
}

// driveState implements the climater api
func (v *Tesla) driveState() (interface{}, error) {
	return v.vehicle.DriveState()
}

// SoC implements the api.Vehicle interface
func (v *Tesla) SoC() (float64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		return float64(res.BatteryLevel), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Tesla)(nil)

// Status implements the api.ChargeState interface
func (v *Tesla) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		if res.ChargingState == "Stopped" || res.ChargingState == "NoPower" || res.ChargingState == "Complete" {
			status = api.StatusB
		}
		if res.ChargingState == "Charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.ChargeRater = (*Tesla)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		return res.ChargeEnergyAdded, nil
	}

	return 0, err
}

const kmPerMile = 1.609344

var _ api.VehicleRange = (*Tesla)(nil)

// Range implements the api.VehicleRange interface
func (v *Tesla) Range() (int64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		// miles to km
		return int64(kmPerMile * res.BatteryRange), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Tesla)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Tesla) Odometer() (float64, error) {
	res, err := v.vehicleStateG()

	if res, ok := res.(*tesla.VehicleState); err == nil && ok {
		// miles to km
		return kmPerMile * res.Odometer, nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Tesla)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Tesla) FinishTime() (time.Time, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		t := time.Now()
		return t.Add(time.Duration(res.MinutesToFullCharge) * time.Minute), err
	}

	return time.Time{}, err
}

// TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

var _ api.VehiclePosition = (*Tesla)(nil)

// Position implements the api.VehiclePosition interface
func (v *Tesla) Position() (float64, float64, error) {
	res, err := v.driveStateG()
	if res, ok := res.(*tesla.DriveState); err == nil && ok {
		return res.Latitude, res.Longitude, nil
	}

	return 0, 0, err
}

var _ api.VehicleStartCharge = (*Tesla)(nil)

// StartCharge implements the api.VehicleStartCharge interface
func (v *Tesla) StartCharge() error {
	err := v.vehicle.StartCharging()

	if err != nil && err.Error() == "408 Request Timeout" {
		if _, err := v.vehicle.Wakeup(); err != nil {
			return err
		}

		timer := time.NewTimer(90 * time.Second)

		for {
			select {
			case <-timer.C:
				return api.ErrTimeout
			default:
				time.Sleep(2 * time.Second)
				if err := v.vehicle.StartCharging(); err == nil || err.Error() != "408 Request Timeout" {
					return err
				}
			}
		}
	}

	return err
}

var _ api.VehicleStopCharge = (*Tesla)(nil)

// StopCharge implements the api.VehicleStopCharge interface
func (v *Tesla) StopCharge() error {
	err := v.vehicle.StopCharging()

	// ignore sleeping vehicle
	if err != nil && err.Error() == "408 Request Timeout" {
		err = nil
	}

	return err
}
