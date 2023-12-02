package coordinator

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

type dummy struct{}

// NewDummy creates a dummy coordinator without vehicles
func NewDummy() API {
	return new(dummy)
}

func (a *dummy) GetVehicles() []api.Vehicle {
	return nil
}

func (a *dummy) Owner(api.Vehicle) loadpoint.API {
	return nil
}

func (a *dummy) Acquire(api.Vehicle) {}

func (a *dummy) Release(api.Vehicle) {}

func (a *dummy) IdentifyVehicleByStatus() api.Vehicle {
	return nil
}
