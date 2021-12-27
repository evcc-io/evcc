package core

import "github.com/evcc-io/evcc/api"

// configProvider gives access to configuration repository
type configProvider interface {
	Meter(string) api.Meter
	Charger(string) api.Charger
	Switch1p3p(string) api.ChargePhases
	Vehicle(string) api.Vehicle
	Simulator(string) api.Updateable
}
