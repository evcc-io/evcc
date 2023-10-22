package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
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

func Settings(v api.Vehicle) API {
	if dev := Device(v); dev != nil {
		return adapt(dev)
	}
	return nil
}

func adapt(dev config.Device[api.Vehicle]) API {
	return &adapter{fmt.Sprintf("vehicle.%s.", dev.Config().Name)}
}

type adapter struct {
	key string
}

func (v *adapter) GetMinSoc() int {
	if v, err := settings.Int(v.key + "minSoc"); err == nil {
		return int(v)
	}
	return 0
}

func (v *adapter) SetMinSoc(soc int) {
	settings.SetInt(v.key+"minSoc", int64(soc))
}

// GetPlanTime returns the plan time
func (v *adapter) GetPlanTime() time.Time {
	if v, err := settings.Time(v.key + "planTime"); err == nil {
		return v
	}
	return time.Time{}
}

// GetPlanSoc returns the charge plan soc
func (v *adapter) GetPlanSoc() float64 {
	if v, err := settings.Float(v.key + "planSoc"); err == nil {
		return v
	}
	return 0
}

// SetPlanSoc sets the charge plan soc
func (v *adapter) SetPlanSoc(finishAt time.Time, soc float64) error {
	if !finishAt.IsZero() && finishAt.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	settings.SetTime(v.key+"planTime", finishAt)
	settings.SetFloat(v.key+"planSoc", soc)

	return nil
}
