package vehicle

import (
	"errors"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/jsgoecke/tesla"
	"gopkg.in/ini.v1"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle        *tesla.Vehicle
	chargeStateG   func() (float64, error)
	chargedEnergyG func() (float64, error)
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

// NewTeslaFromConfig creates a new Tesla vehicle
func NewTeslaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		User, Password         string
		VIN                    string
		Cache                  time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Tesla{
		embed: &embed{cc.Title, cc.Capacity},
	}

	if cc.ClientID == "" {
		// https://tesla-api.timdorr.com/api-basics/authentication
		cc.ClientID, cc.ClientSecret = v.downloadClientID("https://pastebin.com/raw/pS7Z6yyP")
	}

	client, err := tesla.NewClient(&tesla.Auth{
		ClientID:     cc.ClientID,
		ClientSecret: cc.ClientSecret,
		Email:        cc.User,
		Password:     cc.Password,
	})
	if err != nil {
		return nil, err
	}

	vehicles, err := client.Vehicles()
	if err != nil {
		return nil, err
	}

	if cc.VIN == "" && len(vehicles) == 1 {
		v.vehicle = vehicles[0].Vehicle
	} else {
		for _, vehicle := range vehicles {
			if vehicle.Vin == cc.VIN {
				v.vehicle = vehicle.Vehicle
			}
		}
	}

	if v.vehicle == nil {
		return nil, errors.New("vin not found")
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	v.chargedEnergyG = provider.NewCached(v.chargedEnergy, cc.Cache).FloatGetter()

	return v, nil
}

// download client id and secret
func (v *Tesla) downloadClientID(uri string) (string, string) {
	resp, err := http.Get(uri)
	if err == nil {
		defer resp.Body.Close()

		cfg, err := ini.Load(resp.Body)
		if err == nil {
			id := cfg.Section("").Key("TESLA_CLIENT_ID").String()
			secret := cfg.Section("").Key("TESLA_CLIENT_SECRET").String()
			return id, secret
		}
	}

	return "", ""
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Tesla) chargeState() (float64, error) {
	state, err := v.vehicle.ChargeState()
	if err != nil {
		return 0, err
	}
	return float64(state.BatteryLevel), nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Tesla) ChargeState() (float64, error) {
	return v.chargeStateG()
}

// chargedEnergy implements the ChargeRater.ChargedEnergy interface
func (v *Tesla) chargedEnergy() (float64, error) {
	state, err := v.vehicle.ChargeState()
	if err != nil {
		return 0, err
	}
	return state.ChargeEnergyAdded, nil
}

// ChargedEnergy implements the ChargeRater.ChargedEnergy interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	return v.chargedEnergyG()
}

// depends on https://github.com/jsgoecke/tesla/issues/28
//
// CurrentPower implements the ChargeRater.CurrentPower interface
// func (v *Tesla) CurrentPower() (float64, error) {
// 	state, err := v.vehicle.ChargeState()
// 	if err != nil {
// 		return 0, err
// 	}
// 	return state.ChargerPower, err
// }
