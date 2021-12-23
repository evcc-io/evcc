package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/mb"
)

// Smart is an api.Vehicle implementation for Smart cars
type Smart struct {
	*embed
	*request.Helper
}

func init() {
	registry.Add("smart", NewSmartFromConfig)
}

// NewSmartFromConfig creates a new vehicle
func NewSmartFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("smart").Redact(cc.User, cc.Password, cc.VIN)

	v := &Smart{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
	}

	identity := mb.NewIdentity(log)
	err := identity.Login(cc.User, cc.Password)

	return v, err
}

// SoC implements the api.Vehicle interface
func (v *Smart) SoC() (float64, error) {

	return 0, nil
}

// var _ api.ChargeState = (*Smart)(nil)

// // Status implements the api.ChargeState interface
// func (v *Smart) Status() (api.ChargeStatus, error) {
// 	status := api.StatusA // disconnected
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		if res.ChargingState == "Stopped" || res.ChargingState == "NoPower" || res.ChargingState == "Complete" {
// 			status = api.StatusB
// 		}
// 		if res.ChargingState == "Charging" {
// 			status = api.StatusC
// 		}
// 	}

// 	return status, err
// }

// var _ api.ChargeRater = (*Smart)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (v *Smart) ChargedEnergy() (float64, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		return res.ChargeEnergyAdded, nil
// 	}

// 	return 0, err
// }

// const kmPerMile = 1.609344

// var _ api.VehicleRange = (*Smart)(nil)

// // Range implements the api.VehicleRange interface
// func (v *Smart) Range() (int64, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		// miles to km
// 		return int64(kmPerMile * res.BatteryRange), nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleOdometer = (*Smart)(nil)

// // Odometer implements the api.VehicleOdometer interface
// func (v *Smart) Odometer() (float64, error) {
// 	res, err := v.vehicleStateG()

// 	if res, ok := res.(*Smart.VehicleState); err == nil && ok {
// 		// miles to km
// 		return kmPerMile * res.Odometer, nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleFinishTimer = (*Smart)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Smart) FinishTime() (time.Time, error) {
// 	res, err := v.chargeStateG()

// 	if res, ok := res.(*Smart.ChargeState); err == nil && ok {
// 		t := time.Now()
// 		return t.Add(time.Duration(res.MinutesToFullCharge) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

// // TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

// var _ api.VehiclePosition = (*Smart)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *Smart) Position() (float64, float64, error) {
// 	res, err := v.driveStateG()
// 	if res, ok := res.(*Smart.DriveState); err == nil && ok {
// 		return res.Latitude, res.Longitude, nil
// 	}

// 	return 0, 0, err
// }

// var _ api.VehicleStartCharge = (*Smart)(nil)

// // StartCharge implements the api.VehicleStartCharge interface
// func (v *Smart) StartCharge() error {
// 	err := v.vehicle.StartCharging()

// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		if _, err := v.vehicle.Wakeup(); err != nil {
// 			return err
// 		}

// 		timer := time.NewTimer(90 * time.Second)

// 		for {
// 			select {
// 			case <-timer.C:
// 				return api.ErrTimeout
// 			default:
// 				time.Sleep(2 * time.Second)
// 				if err := v.vehicle.StartCharging(); err == nil || err.Error() != "408 Request Timeout" {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	return err
// }

// var _ api.VehicleStopCharge = (*Smart)(nil)

// // StopCharge implements the api.VehicleStopCharge interface
// func (v *Smart) StopCharge() error {
// 	err := v.vehicle.StopCharging()

// 	// ignore sleeping vehicle
// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		err = nil
// 	}

// 	return err
// }
