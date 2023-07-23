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

// Instances returns the instances of the given devices
func Instances[T any](devices []Device[T]) []T {
	res := make([]T, 0, len(devices))
	for _, dev := range devices {
		res = append(res, dev.Instance())
	}
	return res
}

func AddMeter(dev Device[api.Meter]) error {
	return instance.meters.Add(dev)
}

func DeleteMeter(name string) error {
	return instance.meters.Delete(name)
}

func MeterByName(name string) (Device[api.Meter], int, error) {
	return instance.meters.ByName(name)
}

func Meters() []Device[api.Meter] {
	return instance.meters.devices
}

func AddCharger(dev Device[api.Charger]) error {
	return instance.chargers.Add(dev)
}

func DeleteCharger(name string) error {
	return instance.chargers.Delete(name)
}

func ChargerByName(name string) (Device[api.Charger], int, error) {
	return instance.chargers.ByName(name)
}

func Chargers() []Device[api.Charger] {
	return instance.chargers.devices
}

func AddVehicle(dev Device[api.Vehicle]) error {
	return instance.vehicles.Add(dev)
}

func DeleteVehicle(name string) error {
	return instance.vehicles.Delete(name)
}

func VehicleByName(name string) (Device[api.Vehicle], int, error) {
	return instance.vehicles.ByName(name)
}

func Vehicles() []Device[api.Vehicle] {
	return instance.vehicles.devices
}
