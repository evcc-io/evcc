package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/vehicle"
)

type Vehicles interface {
	// Instances returns the list of vehicles apis
	Instances() []api.Vehicle

	// All returns the list of vehicle adapters
	All() []vehicle.API

	// ByName returns a single vehicle adapter by name
	ByName(string) (vehicle.API, error)
}
