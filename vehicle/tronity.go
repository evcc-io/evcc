package vehicle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/tronity"
	"golang.org/x/oauth2"
)

// Tronity is an api.Vehicle implementation for the Tronity api
type Tronity struct {
	*embed
	*request.Helper
	vid   string
	bulkG func() (interface{}, error)
}

func init() {
	registry.Add("tronity", NewTronityFromConfig)
}

// NewTronityFromConfig creates a new Tronity vehicle
func NewTronityFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
		ClientID, ClientSecret string
		Tokens                 Tokens
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Tokens.Access == "" {
		return nil, errors.New("missing token credentials")
	}

	// authenticated http client with logging injected to the Tronity client
	log := util.NewLogger("tronity")

	v := &Tronity{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
	}

	// cfg := tronity.OAuth2Config(cc.ClientID, cc.ClientSecret)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)
	ts := tronity.OAuth2Config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  cc.Tokens.Access,
		RefreshToken: cc.Tokens.Refresh,
		Expiry:       time.Now(),
	})

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	vehicles, err := v.vehicles()
	if err != nil {
		return nil, err
	}

	if cc.VIN == "" && len(vehicles) == 1 {
		v.vid = vehicles[0].ID
	} else {
		for _, vehicle := range vehicles {
			if vehicle.VIN == strings.ToUpper(cc.VIN) {
				v.vid = vehicle.ID
			}
		}
	}

	if v.vid == "" {
		return nil, errors.New("vin not found")
	}

	v.bulkG = provider.NewCached(v.bulk, cc.Cache).InterfaceGetter()

	return v, nil
}

// vehicles implements the vehicles api
func (v *Tronity) vehicles() ([]tronity.Vehicle, error) {
	uri := fmt.Sprintf("%s/v1/vehicles", tronity.URI)

	var res tronity.Vehicles
	err := v.GetJSON(uri, &res)

	return res.Data, err
}

// bulk implements the bulk api
func (v *Tronity) bulk() (interface{}, error) {
	uri := fmt.Sprintf("%s/v1/vehicles/%s/bulk", tronity.URI, v.vid)

	var res tronity.Bulk
	err := v.GetJSON(uri, &res)

	return res, err
}

// SoC implements the api.Vehicle interface
func (v *Tronity) SoC() (float64, error) {
	res, err := v.bulkG()

	if res, ok := res.(*tronity.Bulk); err == nil && ok {
		return float64(res.Level), nil
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
