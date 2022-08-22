package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// API is the external site API
type API interface {
	Healthy() bool
	LoadPoints() []loadpoint.API

	//
	// battery
	//

	GetBufferSoC() float64
	SetBufferSoC(float64) error
	GetPrioritySoC() float64
	SetPrioritySoC(float64) error

	//
	// power and energy
	//

	GetResidualPower() float64
	SetResidualPower(float64) error

	//
	// vehicles
	//

	// GetVehicles is the list of vehicles
	GetVehicles() []api.Vehicle
}
