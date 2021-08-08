package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

type vehicleCoordinator struct {
	tracked map[api.Vehicle]struct{}
}

var coordinator *vehicleCoordinator

func (lp *vehicleCoordinator) own(vehicle api.Vehicle) {
	if lp.tracked == nil {
		lp.tracked = make(map[api.Vehicle]struct{})
	}

	lp.tracked[vehicle] = struct{}{}
}

func (lp *vehicleCoordinator) disown(vehicle api.Vehicle) {
	delete(lp.tracked, vehicle)
}

func (lp *vehicleCoordinator) availableVehicles(vehicles []api.Vehicle) []api.Vehicle {
	var res []api.Vehicle

	for _, vv := range vehicles {
		if _, ok := vv.(api.ChargeState); ok {
			if _, ok := lp.tracked[vv]; !ok {
				res = append(res, vv)
			}
		}
	}

	return res
}

// find active vehicle by charge state
func (lp *vehicleCoordinator) findActiveVehicleByStatus(log *util.Logger, vehicles []api.Vehicle) api.Vehicle {
	var res api.Vehicle

	available := lp.availableVehicles(vehicles)
	// log.DEBUG.Printf("!!available vehicles: %v", funk.Map(available, func(v api.Vehicle) string {
	// 	return v.Title()
	// }))

	for _, vehicle := range available {
		if vs, ok := vehicle.(api.ChargeState); ok {
			status, err := vs.Status()

			if err != nil {
				log.ERROR.Println("vehicle status:", err)
				continue
			}

			log.DEBUG.Printf("vehicle status: %s (%s)", status, vehicle.Title())

			// vehicle is plugged or charging, so it should be the right one
			if status == api.StatusB || status == api.StatusC {
				if res != nil {
					log.DEBUG.Printf("vehicle status: >1 matches, giving up")
					return nil
				}

				res = vehicle
			}
		}
	}

	return res
}
