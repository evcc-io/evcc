package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/jsgoecke/tesla"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle        *tesla.Vehicle
	chargeStateG   provider.FloatGetter
	chargedEnergyG provider.FloatGetter
}

// NewTeslaFromConfig creates a new Tesla vehicle
func NewTeslaFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		Email, Password        string
		VIN                    string
		Cache                  time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	client, err := tesla.NewClient(&tesla.Auth{
		ClientID:     cc.ClientID,
		ClientSecret: cc.ClientSecret,
		Email:        cc.Email,
		Password:     cc.Password,
	})
	if err != nil {
		log.FATAL.Fatalf("cannot create tesla: %v", err)
	}

	vehicles, err := client.Vehicles()
	if err != nil {
		log.FATAL.Fatalf("cannot create tesla: %v", err)
	}

	v := &Tesla{
		embed: &embed{cc.Title, cc.Capacity},
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
		log.FATAL.Fatal("cannot create tesla: vin not found")
	}

	v.chargeStateG = provider.NewCached(log, v.chargeState, cc.Cache).FloatGetter()
	v.chargedEnergyG = provider.NewCached(log, v.chargedEnergy, cc.Cache).FloatGetter()

	return v
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
