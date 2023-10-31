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
	return nil
}

// Adapter creates a vehicle API adapter
func Adapter(log *util.Logger, dev config.Device[api.Vehicle]) API {
	return &adapter{
		log:     log,
		name:    dev.Config().Name,
		Vehicle: dev.Instance(),
	}
}

type adapter struct {
	log         *util.Logger
	name        string
	api.Vehicle // TODO handle instance updates
}

func (v *adapter) key() string {
	return fmt.Sprintf("vehicle.%s.", v.name)
}

func (v *adapter) Name() string {
	return v.name
}

// GetMinSoc returns the min soc
func (v *adapter) GetMinSoc() int {
	if v, err := settings.Int(v.key() + "minSoc"); err == nil {
		return int(v)
	}
	return 0
}

// SetMinSoc sets the min soc
func (v *adapter) SetMinSoc(soc int) {
	v.log.DEBUG.Printf("set %s min soc: %d", v.name, soc)
	settings.SetInt(v.key()+"minSoc", int64(soc))
}

// GetLimitSoc returns the limit soc
func (v *adapter) GetLimitSoc() int {
	if v, err := settings.Int(v.key() + "limitSoc"); err == nil {
		return int(v)
	}
	return 0
}

// SetLimitSoc sets the limit soc
func (v *adapter) SetLimitSoc(soc int) {
	v.log.DEBUG.Printf("set %s limit soc: %d", v.name, soc)
	settings.SetInt(v.key()+"limitSoc", int64(soc))
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

	v.log.DEBUG.Printf("set %s plan soc/time: %d/%v", v.name, soc, ts.Round(time.Second))
	settings.SetTime(v.key()+"planTime", ts)
	settings.SetInt(v.key()+"planSoc", int64(soc))

	return nil
}
