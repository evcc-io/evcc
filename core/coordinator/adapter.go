package coordinator

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"golang.org/x/exp/slices"
)

type adapter struct {
	lp loadpoint.API
	c  *Coordinator
}

// NewAdapter exposes the coordinator for a given loadpoint.
// Using an adapter simplifies the method signatures seen from the loadpoint.
func NewAdapter(lp loadpoint.API, c *Coordinator) API {
	return &adapter{
		lp: lp,
		c:  c,
	}
}

func (a *adapter) GetVehicles() []api.Vehicle {
	return a.c.GetVehicles()
}

func (a *adapter) GetVehicleIndex(v api.Vehicle) int {
	return slices.Index(a.c.vehicles, v)
}

func (a *adapter) Acquire(v api.Vehicle) {
	a.c.acquire(a.lp, v)
}

func (a *adapter) Release(v api.Vehicle) {
	a.c.release(v)
}

func (a *adapter) IdentifyVehicleByStatus() api.Vehicle {
	available := a.c.availableDetectibleVehicles(a.lp)
	return a.c.identifyVehicleByStatus(available)
}
