package coordinator

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// Coordinator coordinates vehicle access between loadpoints
type Coordinator struct {
	log      *util.Logger
	vehicles []api.Vehicle
	tracked  map[api.Vehicle]loadpoint.API
}

// New creates a coordinator for a set of vehicles
func New(log *util.Logger, vehicles []api.Vehicle) *Coordinator {
	return &Coordinator{
		log:      log,
		vehicles: vehicles,
		tracked:  make(map[api.Vehicle]loadpoint.API),
	}
}

func (c *Coordinator) GetVehicles() []api.Vehicle {
	return c.vehicles
}

func (c *Coordinator) acquire(owner loadpoint.API, vehicle api.Vehicle) {
	if o, ok := c.tracked[vehicle]; ok && o != owner {
		o.SetVehicle(nil)
	}
	c.tracked[vehicle] = owner
}

func (c *Coordinator) release(vehicle api.Vehicle) {
	delete(c.tracked, vehicle)
}

// availableDetectibleVehicles is the list of vehicles that are currently not
// associated to another loadpoint and have a status api that allows for detection
func (c *Coordinator) availableDetectibleVehicles(owner loadpoint.API, includeIdCapable bool) []api.Vehicle {
	var res []api.Vehicle

	for _, vv := range c.vehicles {
		// status api available
		if _, ok := vv.(api.ChargeState); ok {
			// available or associated to current loadpoint
			if o, ok := c.tracked[vv]; o == owner || !ok {
				// no identifiers configured or identifiers ignored
				if includeIdCapable || len(vv.Identifiers()) == 0 {
					res = append(res, vv)
				}
			}
		}
	}

	return res
}

// identifyVehicleByStatus finds active vehicle by charge state
func (c *Coordinator) identifyVehicleByStatus(available []api.Vehicle) api.Vehicle {
	var res api.Vehicle
	for _, vehicle := range available {
		if vs, ok := vehicle.(api.ChargeState); ok {
			status, err := vs.Status()
			if err != nil {
				c.log.ERROR.Println("vehicle status:", err)
				continue
			}

			c.log.DEBUG.Printf("vehicle status: %s (%s)", status, vehicle.Title())

			// vehicle is plugged or charging, so it should be the right one
			if status == api.StatusB || status == api.StatusC {
				if res != nil {
					c.log.WARN.Println("vehicle status: >1 matches, giving up")
					return nil
				}

				res = vehicle
			}
		}
	}

	return res
}
