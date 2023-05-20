package config

import (
	"github.com/evcc-io/evcc/api"
)

type provider struct {
	meters   handler[api.Meter]
	chargers handler[api.Charger]
	vehicles handler[api.Vehicle]
}

var instance = new(provider)

func init() {
	instance.meters.visited = make(map[string]bool)
	instance.chargers.visited = make(map[string]bool)
	instance.vehicles.visited = make(map[string]bool)
}

func TrackVisitors() {
	instance.meters.TrackVisitors()
}

func AddMeter(conf Named, meter api.Meter) error {
	return instance.meters.Add(conf, meter)
}

func MeterByName(name string) (api.Meter, int, error) {
	return instance.meters.ByName(name)
}

func Meters() []api.Meter {
	return instance.meters.Slice()
}

func MetersMap() map[string]api.Meter {
	return instance.meters.Map()
}

func MetersConfig() []Named {
	return instance.meters.Config()
}

func AddCharger(conf Named, charger api.Charger) error {
	return instance.chargers.Add(conf, charger)
}

func ChargerByName(name string) (api.Charger, int, error) {
	return instance.chargers.ByName(name)
}

func Chargers() []api.Charger {
	return instance.chargers.Slice()
}

func ChargersMap() map[string]api.Charger {
	return instance.chargers.Map()
}

func ChargersConfig() []Named {
	return instance.chargers.Config()
}

func AddVehicle(conf Named, vehicle api.Vehicle) error {
	return instance.vehicles.Add(conf, vehicle)
}

func VehicleByName(name string) (api.Vehicle, int, error) {
	return instance.vehicles.ByName(name)
}

func Vehicles() []api.Vehicle {
	return instance.vehicles.Slice()
}

func VehiclesMap() map[string]api.Vehicle {
	return instance.vehicles.Map()
}

func VehiclesConfig() []Named {
	return instance.vehicles.Config()
}
