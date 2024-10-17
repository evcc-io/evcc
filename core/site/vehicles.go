package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/vehicle"
)

type Vehicles interface {
	// Settings returns the list of vehicle adapters
	Settings() []vehicle.API

	// ByName returns a single vehicle adapter by name
	ByName(string) (vehicle.API, error)

	// All returns the list of vehicle instances
	Instances() []api.Vehicle
}
