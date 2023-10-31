package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
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

	site.publish(keys.Vehicles, vehicleTitles(site.Vehicles().Instances()))
}

var _ site.Vehicles = (*vehicles)(nil)

type vehicles struct {
	log *util.Logger
}

func (vv *vehicles) Instances() []api.Vehicle {
	devs := config.Vehicles().Devices()

	res := make([]api.Vehicle, 0, len(devs))
	for _, dev := range devs {
		res = append(res, dev.Instance())
	}

	return res
}

func (vv *vehicles) All() []vehicle.API {
	devs := config.Vehicles().Devices()

	res := make([]vehicle.API, 0, len(devs))
	for _, dev := range devs {
		res = append(res, vehicle.Adapter(vv.log, dev))
	}

	return res
}

func (vv *vehicles) ByName(name string) (vehicle.API, error) {
	dev, err := config.Vehicles().ByName(name)
	if err != nil {
		return nil, err
	}

	return vehicle.Adapter(vv.log, dev), nil
}
