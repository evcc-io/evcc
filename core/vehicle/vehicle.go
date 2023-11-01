package vehicle

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

func device(vehicle api.Vehicle) config.Device[api.Vehicle] {
	for _, dev := range config.Vehicles().Devices() {
		if dev.Instance() == vehicle {
			return dev
		}
	}
	return nil
}

func Settings(log *util.Logger, v api.Vehicle) API {
	if dev := device(v); dev != nil {
		return Adapter(log, dev)
	}
	return &dummy{v}
}

// Adapter creates a vehicle API adapter
func Adapter(log *util.Logger, dev config.Device[api.Vehicle]) API {
	return &adapter{
		log:     log,
		name:    dev.Config().Name,
		Vehicle: dev.Instance(),
	}
}
