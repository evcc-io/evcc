package config

import (
	"github.com/evcc-io/evcc/api"
)

var instance = struct {
	meters   *handler[api.Meter]
	chargers *handler[api.Charger]
	vehicles *handler[api.Vehicle]
}{
	meters:   new(handler[api.Meter]),
	chargers: new(handler[api.Charger]),
	vehicles: new(handler[api.Vehicle]),
}

type Handler[T any] interface {
	Devices() []Device[T]
	Add(dev Device[T]) error
	Delete(name string) error
	ByName(name string) (Device[T], error)
}

func Meters() Handler[api.Meter] {
	return instance.meters
}

func Chargers() Handler[api.Charger] {
	return instance.chargers
}

func Vehicles() Handler[api.Vehicle] {
	return instance.vehicles
}

// Instances returns the instances of the given devices
func Instances[T any](devices []Device[T]) []T {
	res := make([]T, 0, len(devices))
	for _, dev := range devices {
		res = append(res, dev.Instance())
	}
	return res
}
