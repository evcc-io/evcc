package core

import "github.com/evcc-io/evcc/api"

// configProvider gives access to configuration repository
type configProvider interface {
	Meter(string) api.Meter
	Charger(string) api.Charger
	Vehicle(string) api.Vehicle
}
