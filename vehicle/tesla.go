package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/jsgoecke/tesla"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle        *tesla.Vehicle
	chargeStateG   *provider.CacheGetter
	chargedEnergyG *provider.CacheGetter
}

// NewTeslaFromConfig creates a new Tesla vehicle
func NewTeslaFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                                   string
		Capacity                                int64
		ClientID, ClientSecret, Email, Password string
		Vehicle                                 int
		Cache                                   time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

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
		embed:   &embed{cc.Title, cc.Capacity},
		vehicle: vehicles[0].Vehicle,
	}

	v.chargeStateG = provider.NewCacheGetter(v.chargeState, cc.Cache)
	v.chargedEnergyG = provider.NewCacheGetter(v.chargedEnergy, cc.Cache)

	return v
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Tesla) chargeState() (float64, error) {
	state, err := v.vehicle.ChargeState()
	return float64(state.BatteryLevel), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Tesla) ChargeState() (float64, error) {
	return v.chargeStateG.FloatGetter()
}

// chargedEnergy implements the ChargeRater.ChargedEnergy interface
func (v *Tesla) chargedEnergy() (float64, error) {
	state, err := v.vehicle.ChargeState()
	return state.ChargeEnergyAdded, err
}

// ChargedEnergy implements the ChargeRater.ChargedEnergy interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	return v.chargedEnergyG.FloatGetter()
}

// depends on https://github.com/jsgoecke/tesla/issues/28
//
// CurrentPower implements the ChargeRater.CurrentPower interface
// func (v *Tesla) CurrentPower() (float64, error) {
// 	state, err := v.vehicle.ChargeState()
// 	return state.ChargerPower, err
// }
