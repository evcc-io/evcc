package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
)

// vehicleTitles returns a list of vehicle titles
func vehicleTitles(vehicles []api.Vehicle) []string {
	return lo.Map(vehicles, func(v api.Vehicle, _ int) string {
		return v.Title()
	})
}

// updateVehicles adds or removes a vehicle asynchronously
func (site *Site) updateVehicles(op config.Operation, dev config.Device[api.Vehicle]) {
	vehicle := dev.Instance()

	switch op {
	case config.OpAdd:
		site.coordinator.Add(vehicle)

	case config.OpDelete:
		site.coordinator.Delete(vehicle)
	}

	site.publish("vehicles", vehicleTitles(site.GetVehicles()))
}
