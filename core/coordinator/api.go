package coordinator

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// API is the coordinator API
type API interface {
	// GetVehicles returns the list of all vehicles
	GetVehicles() []api.Vehicle

	// Owner returns the loadpoint that currently owns the vehicle
	Owner(api.Vehicle) loadpoint.API

	// Aquire acquires the vehicle for the loadpoint and releases it at any other loadpoint
	Acquire(api.Vehicle)

	// Release releases a vehicle from a loadpoint
	Release(api.Vehicle)

	// IdentifyVehicleByStatus returns an available vehicle that is currently connected or charging
	IdentifyVehicleByStatus() api.Vehicle
}
