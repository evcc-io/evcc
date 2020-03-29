package vehicle

import (
	"github.com/andig/evcc/api"
	"github.com/jsgoecke/tesla"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle *tesla.Vehicle
}

// NewTeslaFromConfig creates a new Tesla vehicle
func NewTeslaFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                                   string
		Capacity                                int64
		ClientID, ClientSecret, Email, Password string
		Vehicle                                 int
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

	return &Tesla{
		embed:   &embed{cc.Title, cc.Capacity},
		vehicle: vehicles[0].Vehicle,
	}
}

// ChargeState implements the Vehicle.ChargeState interface
func (m *Tesla) ChargeState() (float64, error) {
	state, err := m.vehicle.ChargeState()
	return float64(state.BatteryLevel), err
}

// depends on https://github.com/jsgoecke/tesla/issues/28
//
// CurrentPower implements the ChargeRater.CurrentPower interface
// func (m *Tesla) CurrentPower() (float64, error) {
// 	state, err := m.vehicle.ChargeState()
// 	return state.ChargerPower, err
// }

// ChargedEnergy implements the ChargeRater.ChargedEnergy interface
func (m *Tesla) ChargedEnergy() (float64, error) {
	state, err := m.vehicle.ChargeState()
	return state.ChargeEnergyAdded, err
}
