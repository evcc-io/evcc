package site

import (
	"github.com/evcc-io/evcc/core/vehicle"
)

type Vehicles interface {
	// All returns the list of vehicle adapters
	All() []vehicle.API

	// ByName returns a single vehicle adapter by name
	ByName(string) (vehicle.API, error)
}
