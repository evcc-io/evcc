package soc

import (
	"github.com/andig/evcc/api"
	"github.com/jsgoecke/tesla"
)

// Tesla is an api.SoC implementation for Tesla cars
type Tesla struct {
	capacity int64
	title    string
	client   *tesla.Client
	vehicle  *tesla.Vehicle
}

// NewTeslaFromConfig creates a new SoC
func NewTeslaFromConfig(log *api.Logger, title string, other map[string]interface{}) api.SoC {
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
		capacity: cc.Capacity,
		title:    cc.Title,
		client:   client,
		vehicle:  vehicles[0].Vehicle,
	}
}

// Title implements the SoC.Title interface
func (m *Tesla) Title() string {
	return m.title
}

// Capacity implements the SoC.Capacity interface
func (m *Tesla) Capacity() int64 {
	return m.capacity
}

// ChargeState implements the SoC.ChargeState interface
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
