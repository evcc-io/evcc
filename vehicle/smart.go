package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/mb"
)

// Smart is an api.Vehicle implementation for Smart cars
type Smart struct {
	*embed
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
		embed: &cc.embed,
	}

	identity := mb.NewIdentity(log)
	err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := mb.NewAPI(log, identity)

	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	err = api.Status(cc.VIN)

	return v, err
}

// SoC implements the api.Vehicle interface
func (v *Smart) SoC() (float64, error) {
	return 0, api.ErrNotAvailable
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
