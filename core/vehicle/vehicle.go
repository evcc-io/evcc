package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

// TODO logging

var Handler config.Handler[api.Vehicle]

func Device(vehicle api.Vehicle) config.Device[api.Vehicle] {
	if Handler == nil {
		return nil
	}
	for _, dev := range Handler.Devices() {
		if dev.Instance() == vehicle {
			return dev
		}
	}
	return nil
}

var log = util.NewLogger("vehicle")

func Settings(v api.Vehicle) API {
	if dev := Device(v); dev != nil {
		return Adapter(dev)
	}
	return nil
}

// Adapter creates a vehicle API adapter
func Adapter(dev config.Device[api.Vehicle]) API {
	return &adapter{
		name: dev.Config().Name,
	}
}

type adapter struct {
	name string
}

// TODO limitSoc handler

func (v *adapter) key() string {
	return fmt.Sprintf("vehicle.%s.", v.name)
}

func (v *adapter) GetMinSoc() int {
	if v, err := settings.Int(v.key() + "minSoc"); err == nil {
		return int(v)
	}
	return 0
}

func (v *adapter) SetMinSoc(soc int) {
	log.DEBUG.Printf("set %s min soc: %d", v.name, soc)
	settings.SetInt(v.key()+"minSoc", int64(soc))
}

// GetPlanTime returns the plan time
func (v *adapter) GetPlanTime() time.Time {
	if v, err := settings.Time(v.key() + "planTime"); err == nil {
		return v
	}
	return time.Time{}
}

// GetPlanSoc returns the charge plan soc
func (v *adapter) GetPlanSoc() int {
	if v, err := settings.Int(v.key() + "planSoc"); err == nil {
		return int(v)
	}
	return 0
}

// SetPlanSoc sets the charge plan soc
func (v *adapter) SetPlanSoc(ts time.Time, soc int) error {
	if !ts.IsZero() && ts.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	log.DEBUG.Printf("set %s plan soc/time: %d/%v", v.name, soc, ts.Round(time.Second))
	settings.SetTime(v.key()+"planTime", ts)
	settings.SetInt(v.key()+"planSoc", int64(soc))

	return nil
}
