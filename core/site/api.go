package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// API is the external site API
type API interface {
	Healthy() bool
	Loadpoints() []loadpoint.API

	//
	// battery
	//

	GetBufferSoc() float64
	SetBufferSoc(float64) error
	GetPrioritySoc() float64
	SetPrioritySoc(float64) error

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
