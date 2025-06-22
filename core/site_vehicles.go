package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
)

type planStruct struct {
	Soc          int       `json:"soc"`
	Precondition int64     `json:"precondition"`
	Time         time.Time `json:"time"`
}

type vehicleStruct struct {
	Title          string                    `json:"title"`
	Icon           string                    `json:"icon,omitempty"`
	Capacity       float64                   `json:"capacity,omitempty"`
	Phases         int                       `json:"phases,omitempty"`
	MinSoc         int                       `json:"minSoc,omitempty"`
	LimitSoc       int                       `json:"limitSoc,omitempty"`
	MinCurrent     float64                   `json:"minCurrent,omitempty"`
	MaxCurrent     float64                   `json:"maxCurrent,omitempty"`
	Priority       int                       `json:"priority,omitempty"`
	Features       []string                  `json:"features,omitempty"`
	Plan           *planStruct               `json:"plan,omitempty"`
	RepeatingPlans []api.RepeatingPlanStruct `json:"repeatingPlans"`
}

// publishVehicles returns a list of vehicle titles
func (site *Site) publishVehicles() {
	vv := site.Vehicles().Settings()
	res := make(map[string]vehicleStruct, len(vv))

	for _, v := range vv {
		var plan *planStruct

		if time, precondition, soc := v.GetPlanSoc(); !time.IsZero() {
			plan = &planStruct{Soc: soc, Precondition: int64(precondition.Seconds()), Time: time}
		}

		instance := v.Instance()
		ac := instance.OnIdentified()

		res[v.Name()] = vehicleStruct{
			Title:          instance.GetTitle(),
			Icon:           instance.Icon(),
			Capacity:       instance.Capacity(),
			Phases:         instance.Phases(),
			MinSoc:         v.GetMinSoc(),
			LimitSoc:       v.GetLimitSoc(),
			MinCurrent:     ac.MinCurrent,
			MaxCurrent:     ac.MaxCurrent,
			Priority:       ac.Priority,
			Features:       lo.Map(instance.Features(), func(f api.Feature, _ int) string { return f.String() }),
			Plan:           plan,
			RepeatingPlans: v.GetRepeatingPlans(),
		}

		if lp := site.coordinator.Owner(instance); lp != nil {
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
