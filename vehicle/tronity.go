package vehicle

import (
	"context"
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// Tronity is an api.Vehicle implementation for the Tronity api
type Tronity struct {
	*embed
	chargeStateG  func() (interface{}, error)
	climateStateG func() (interface{}, error)
}

func init() {
	registry.Add("tronity", NewTronityFromConfig)
}

// NewTronityFromConfig creates a new Tronity vehicle
func NewTronityFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string // deprecated
		Tokens         Tokens
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Tokens.Access == "" {
		return nil, errors.New("missing token credentials")
	}

	v := &Tronity{
		embed: &cc.embed,
	}

	// authenticated http client with logging injected to the Tronity client
	log := util.NewLogger("tronity")
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)

	options := []Tronity.ClientOption{Tronity.WithToken(&oauth2.Token{
		AccessToken:  cc.Tokens.Access,
		RefreshToken: cc.Tokens.Refresh,
		Expiry:       time.Now(),
	})}

	client, err := Tronity.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	// vehicles, err := client.Vehicles()
	// if err != nil {
	// 	return nil, err
	// }

	// if cc.VIN == "" && len(vehicles) == 1 {
	// 	v.vehicle = vehicles[0]
	// } else {
	// 	for _, vehicle := range vehicles {
	// 		if vehicle.Vin == strings.ToUpper(cc.VIN) {
	// 			v.vehicle = vehicle
	// 		}
	// 	}
	// }

	// if v.vehicle == nil {
	// 	return nil, errors.New("vin not found")
	// }

	// v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).InterfaceGetter()
	// v.climateStateG = provider.NewCached(v.climateState, cc.Cache).InterfaceGetter()

	return v, nil
}

// // chargeState implements the charge state api
// func (v *Tronity) chargeState() (interface{}, error) {
// 	return v.ChargeState()
// }

// // climateState implements the climater api
// func (v *Tronity) climateState() (interface{}, error) {
// 	return v.ClimateState()
// }

// SoC implements the api.Vehicle interface
func (v *Tronity) SoC() (float64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*Tronity.ChargeState); err == nil && ok {
		return float64(res.BatteryLevel), nil
	}

	return 0, err
}

// var _ api.ChargeState = (*Tronity)(nil)

// // Status implements the api.ChargeState interface
// func (v *Tronity) Status() (api.ChargeStatus, error) {
// 	status := api.StatusA // disconnected
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Tronity.ChargeState); err == nil && ok {
// 		if res.ChargingState == "Stopped" || res.ChargingState == "NoPower" || res.ChargingState == "Complete" {
// 			status = api.StatusB
// 		}
// 		if res.ChargingState == "Charging" {
// 			status = api.StatusC
// 		}
// 	}

// 	return status, err
// }

// var _ api.ChargeRater = (*Tronity)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (v *Tronity) ChargedEnergy() (float64, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Tronity.ChargeState); err == nil && ok {
// 		return float64(res.ChargeEnergyAdded), nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleRange = (*Tronity)(nil)

// // Range implements the api.VehicleRange interface
// func (v *Tronity) Range() (int64, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Tronity.ChargeState); err == nil && ok {
// 		// miles to km
// 		return int64(1.609344 * res.EstBatteryRange), nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleFinishTimer = (*Tronity)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Tronity) FinishTime() (time.Time, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Tronity.ChargeState); err == nil && ok {
// 		t := time.Now()
// 		return t.Add(time.Duration(res.MinutesToFullCharge) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

// // TODO api.Climater implementation has been removed as it drains battery. Re-check at t later time.

// var _ api.VehicleStartCharge = (*Tronity)(nil)

// // StartCharge implements the api.VehicleStartCharge interface
// func (v *Tronity) StartCharge() error {
// 	err := v.vehicle.StartCharging()

// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		if _, err := v.vehicle.Wakeup(); err != nil {
// 			return err
// 		}

// 		timer := time.NewTimer(90 * time.Second)

// 		for {
// 			select {
// 			case <-timer.C:
// 				return api.ErrTimeout
// 			default:
// 				time.Sleep(2 * time.Second)
// 				if err := v.vehicle.StartCharging(); err == nil || err.Error() != "408 Request Timeout" {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	return err
// }

// var _ api.VehicleStopCharge = (*Tronity)(nil)

// // StopCharge implements the api.VehicleStopCharge interface
// func (v *Tronity) StopCharge() error {
// 	err := v.vehicle.StopCharging()

// 	// ignore sleeping vehicle
// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		err = nil
// 	}

// 	return err
// }
