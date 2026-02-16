package config

import (
	evbus "github.com/asaskevich/EventBus"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/hems/hems"
)

var bus = evbus.New()

var instance struct {
	meters     *handler[api.Meter]
	chargers   *handler[api.Charger]
	vehicles   *handler[api.Vehicle]
	circuits   *handler[api.Circuit]
	hems       *handler[hems.API]
	loadpoints *handler[loadpoint.API]
}

func init() {
	Reset()
}

func Reset() {
	instance.meters = &handler[api.Meter]{topic: "meter"}
	instance.chargers = &handler[api.Charger]{topic: "charger"}
	instance.vehicles = &handler[api.Vehicle]{topic: "vehicle"}
	instance.circuits = &handler[api.Circuit]{topic: "circuit"}
	instance.loadpoints = &handler[loadpoint.API]{topic: "loadpoint"}
}

type Handler[T any] interface {
	Subscribe(fn func(Operation, Device[T]))
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

func Circuits() Handler[api.Circuit] {
	return instance.circuits
}

func Hems() Handler[hems.API] {
	return instance.hems
}

func Loadpoints() Handler[loadpoint.API] {
	return instance.loadpoints
}

// Instances returns the instances of the given devices
func Instances[T any](devices []Device[T]) []T {
	res := make([]T, 0, len(devices))
	for _, dev := range devices {
		res = append(res, dev.Instance())
	}
	return res
}
