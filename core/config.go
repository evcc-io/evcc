package core

import "github.com/evcc-io/evcc/api"

// configProvider gives access to configuration repository
type configProvider interface {
	Meter(string) (api.Meter, error)
	Charger(string) (api.Charger, error)
	Vehicle(string) (api.Vehicle, error)
}
