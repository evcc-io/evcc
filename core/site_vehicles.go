package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type planStruct struct {
	Soc  int       `json:"soc"`
	Time time.Time `json:"time"`
}

type vehicleStruct struct {
	Title    string       `json:"title"`
	MinSoc   int          `json:"minSoc,omitempty"`
	LimitSoc int          `json:"limitSoc,omitempty"`
	Plans    []planStruct `json:"plans,omitempty"`
}

// publishVehicles returns a list of vehicle titles
func (site *Site) publishVehicles() {
	vv := site.Vehicles().Settings()
	res := make(map[string]vehicleStruct, len(vv))

	for _, v := range vv {
		var plans []planStruct

		// TODO: add support for multiple plans
		if time, soc := v.GetPlanSoc(); !time.IsZero() {
			plans = append(plans, planStruct{Soc: soc, Time: time})
		}

		res[v.Name()] = vehicleStruct{
			Title:    v.Instance().Title(),
			MinSoc:   v.GetMinSoc(),
			LimitSoc: v.GetLimitSoc(),
			Plans:    plans,
		}

		if lp := site.coordinator.Owner(v.Instance()); lp != nil {
			lp.PublishEffectiveValues()
		}
	}

	site.publish(keys.Vehicles, res)
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

	// TODO remove vehicle from mqtt
	site.publishVehicles()
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

func (vv *vehicles) Settings() []vehicle.API {
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
