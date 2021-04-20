package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const (
	ovmsDextersWebApi = "https://dexters-web.de/api/call?fn.name=ovms/export&fn.vehicleid=%s&fn.carpass=%s&fn.last=1&fn.format=csv"
)

// OVMS is an api.Vehicle implementation for dexters-web server requests
type OVMS struct {
	*embed
	*request.Helper
	vehicleId, carPassword string
}

func init() {
	registry.Add("ovms", NewOVMSFromConfig)
}

// NewOVMSFromConfig creates a new vehicle
func NewOVMSFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		VehicleID, CarPassword string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ovms")

	v := &OVMS{
		embed:       &embed{cc.Title, cc.Capacity},
		Helper:      request.NewHelper(log),
		vehicleId:   cc.VehicleID,
		carPassword: cc.CarPassword,
	}

	return v, nil
}

// SoC implements the api.Vehicle interface
func (v *OVMS) SoC() (float64, error) {
	// res, err := v.batteryG()

	// if res, ok := res.(kamereonResponse); err == nil && ok {
	// 	return float64(res.Data.Attributes.BatteryLevel), nil
	// }

	return 23, nil
}

var _ api.ChargeState = (*OVMS)(nil)

// Status implements the api.ChargeState interface
func (v *OVMS) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	// res, err := v.batteryG()
	// if res, ok := res.(kamereonResponse); err == nil && ok {
	// 	if res.Data.Attributes.PlugStatus > 0 {
	// 		status = api.StatusB
	// 	}
	// 	if res.Data.Attributes.ChargingStatus >= 1.0 {
	// 		status = api.StatusC
	// 	}
	// }

	return status, nil
}

var _ api.VehicleRange = (*OVMS)(nil)

// Range implements the api.VehicleRange interface
func (v *OVMS) Range() (int64, error) {
	// res, err := v.batteryG()

	// if res, ok := res.(kamereonResponse); err == nil && ok {
	// 	return int64(res.Data.Attributes.BatteryAutonomy), nil
	// }

	return 0, nil
}

var _ api.VehicleFinishTimer = (*OVMS)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *OVMS) FinishTime() (time.Time, error) {
	// res, err := v.batteryG()

	// if res, ok := res.(kamereonResponse); err == nil && ok {
	// 	timestamp, err := time.Parse(time.RFC3339, res.Data.Attributes.Timestamp)

	// 	if res.Data.Attributes.RemainingTime == nil {
	// 		return time.Time{}, api.ErrNotAvailable
	// 	}

	// 	return timestamp.Add(time.Duration(*res.Data.Attributes.RemainingTime) * time.Minute), err
	// }

	return time.Time{}, nil
}

var _ api.VehicleClimater = (*OVMS)(nil)

// Climater implements the api.VehicleClimater interface
func (v *OVMS) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	return false, 0, 0, api.ErrNotAvailable
}
