package vehicle

import (
	"context"
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/teslamotors/vehicle-command/pkg/account"
	"github.com/teslamotors/vehicle-command/pkg/cache"
	"github.com/teslamotors/vehicle-command/pkg/protocol"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
	"golang.org/x/oauth2"
)

// TeslaVC is an api.Vehicle implementation for Tesla cars.
// It uses the official Tesla vehicle-command api.
type TeslaVC struct {
	*embed
	vehicle *vehicle.Vehicle
	// dataG   func() (*tesla.VehicleData, error)
}

func init() {
	registry.Add("tesla-vehicle-command", NewTeslaVCFromConfig)
}

// NewTeslaVCFromConfig creates a new vehicle
func NewTeslaVCFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		Tokens Tokens
		VIN    string
		Cache  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.Tokens.Error(); err != nil {
		return nil, err
	}

	user, err := account.New(cc.Tokens.Access)
	if err != nil {
		return nil, err
	}

	v := &TeslaVC{
		embed: &cc.embed,
	}

	// authenticated http client with logging injected to the TeslaVC client
	log := util.NewLogger("tesla").Redact(cc.Tokens.Access, cc.Tokens.Refresh)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	// privKey, err := protocol.UnmarshalECDHPrivateKey(nil)
	// if err != nil {
	// 	logger.Printf("Failed to load private key: %s", err)
	// 	return
	// }
	privKey := protocol.UnmarshalECDHPrivateKey(nil)
	if privKey == nil {
		return nil, errors.New("failed to load private key")
	}

	v.vehicle, err = user.GetVehicle(ctx, cc.VIN, privKey, cache.New(8))
	if err != nil {
		return nil, err
	}

	// if v.Title_ == "" {
	// 	v.Title_ = v.vehicle.DisplayName
	// }

	// v.dataG = provider.Cached(func() (*tesla.VehicleData, error) {
	// 	res, err := v.vehicle.Data()
	// 	return res, v.apiError(err)
	// }, cc.Cache)

	return v, nil
}

// Soc implements the api.Vehicle interface
func (v *TeslaVC) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.UsableBatteryLevel), nil
}

// var _ api.ChargeState = (*TeslaVC)(nil)

// // Status implements the api.ChargeState interface
// func (v *TeslaVC) Status() (api.ChargeStatus, error) {
// 	status := api.StatusA // disconnected
// 	res, err := v.dataG()
// 	if err != nil {
// 		return status, err
// 	}

// 	switch res.Response.ChargeState.ChargingState {
// 	case "Stopped", "NoPower", "Complete":
// 		status = api.StatusB
// 	case "Charging":
// 		status = api.StatusC
// 	}

// 	return status, nil
// }

// var _ api.ChargeRater = (*TeslaVC)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (v *TeslaVC) ChargedEnergy() (float64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, err
// 	}
// 	return res.Response.ChargeState.ChargeEnergyAdded, nil
// }

// const kmPerMile = 1.609344

// var _ api.VehicleRange = (*TeslaVC)(nil)

// // Range implements the api.VehicleRange interface
// func (v *TeslaVC) Range() (int64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, err
// 	}
// 	// miles to km
// 	return int64(kmPerMile * res.Response.ChargeState.BatteryRange), nil
// }

// var _ api.VehicleOdometer = (*TeslaVC)(nil)

// // Odometer implements the api.VehicleOdometer interface
// func (v *TeslaVC) Odometer() (float64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, err
// 	}
// 	// miles to km
// 	return kmPerMile * res.Response.VehicleState.Odometer, nil
// }

// var _ api.VehicleFinishTimer = (*TeslaVC)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *TeslaVC) FinishTime() (time.Time, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return time.Time{}, err
// 	}
// 	return time.Now().Add(time.Duration(res.Response.ChargeState.MinutesToFullCharge) * time.Minute), nil
// }

// // TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

// var _ api.VehiclePosition = (*TeslaVC)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *TeslaVC) Position() (float64, float64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, 0, err
// 	}
// 	return res.Response.DriveState.Latitude, res.Response.DriveState.Longitude, nil
// }

// var _ api.SocLimiter = (*TeslaVC)(nil)

// // TargetSoc implements the api.SocLimiter interface
// func (v *TeslaVC) TargetSoc() (float64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, err
// 	}
// 	return float64(res.Response.ChargeState.ChargeLimitSoc), nil
// }

// var _ api.CurrentLimiter = (*TeslaVC)(nil)

// // StartCharge implements the api.VehicleChargeController interface
// func (v *TeslaVC) MaxCurrent(current int64) error {
// 	return v.apiError(v.vehicle.SetChargingAmps(int(current)))
// }

// var _ api.Resurrector = (*TeslaVC)(nil)

// func (v *TeslaVC) WakeUp() error {
// 	_, err := v.vehicle.Wakeup()
// 	return v.apiError(err)
// }

// var _ api.VehicleChargeController = (*TeslaVC)(nil)

// // StartCharge implements the api.VehicleChargeController interface
// func (v *TeslaVC) StartCharge() error {
// 	err := v.apiError(v.vehicle.StartCharging())
// 	if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
// 		return nil
// 	}
// 	return err
// }

// // StopCharge implements the api.VehicleChargeController interface
// func (v *TeslaVC) StopCharge() error {
// 	err := v.apiError(v.vehicle.StopCharging())

// 	// ignore sleeping vehicle
// 	if errors.Is(err, api.ErrAsleep) {
// 		err = nil
// 	}

// 	return err
// }
