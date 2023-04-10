package vehicle

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle *tesla.Vehicle
	dataG   func() (*tesla.VehicleData, error)
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

// NewTeslaFromConfig creates a new vehicle
func NewTeslaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Tesla{
		embed: &cc.embed,
	}

	// authenticated http client with logging injected to the Tesla client
	log := util.NewLogger("tesla").Redact(cc.Tokens.Access, cc.Tokens.Refresh)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	options := []tesla.ClientOption{tesla.WithToken(&oauth2.Token{
		AccessToken:  cc.Tokens.Access,
		RefreshToken: cc.Tokens.Refresh,
		Expiry:       time.Now(),
	})}

	client, err := tesla.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	v.vehicle, err = ensureVehicleEx(
		cc.VIN, client.Vehicles,
		func(v *tesla.Vehicle) string {
			return v.Vin
		},
	)

	if err != nil {
		return nil, err
	}

	if v.Title_ == "" {
		v.Title_ = v.vehicle.DisplayName
	}

	v.dataG = provider.Cached(func() (*tesla.VehicleData, error) {
		res, err := v.vehicle.Data()
		return res, v.apiError(err)
	}, cc.Cache)

	return v, nil
}

// apiError converts HTTP 408 error to ErrTimeout
func (v *Tesla) apiError(err error) error {
	if err != nil && err.Error() == "408 Request Timeout" {
		err = api.ErrAsleep
	}
	return err
}

// Soc implements the api.Vehicle interface
func (v *Tesla) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.UsableBatteryLevel), nil
}

var _ api.ChargeState = (*Tesla)(nil)

// Status implements the api.ChargeState interface
func (v *Tesla) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected
	res, err := v.dataG()
	if err != nil {
		return status, err
	}

	switch res.Response.ChargeState.ChargingState {
	case "Stopped", "NoPower", "Complete":
		status = api.StatusB
	case "Charging":
		status = api.StatusC
	}

	return status, nil
}

var _ api.ChargeRater = (*Tesla)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return res.Response.ChargeState.ChargeEnergyAdded, nil
}

const kmPerMile = 1.609344

var _ api.VehicleRange = (*Tesla)(nil)

// Range implements the api.VehicleRange interface
func (v *Tesla) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return int64(kmPerMile * res.Response.ChargeState.BatteryRange), nil
}

var _ api.VehicleOdometer = (*Tesla)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Tesla) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return kmPerMile * res.Response.VehicleState.Odometer, nil
}

var _ api.VehicleFinishTimer = (*Tesla)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Tesla) FinishTime() (time.Time, error) {
	res, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(time.Duration(res.Response.ChargeState.MinutesToFullCharge) * time.Minute), nil
}

// TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

var _ api.VehiclePosition = (*Tesla)(nil)

// Position implements the api.VehiclePosition interface
func (v *Tesla) Position() (float64, float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, 0, err
	}
	return res.Response.DriveState.Latitude, res.Response.DriveState.Longitude, nil
}

var _ api.SocLimiter = (*Tesla)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Tesla) TargetSoc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.ChargeLimitSoc), nil
}

var _ api.CurrentController = (*Tesla)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Tesla) MaxCurrent(current int64) error {
	return v.apiError(v.vehicle.SetChargingAmps(int(current)))
}

var _ api.Resurrector = (*Tesla)(nil)

func (v *Tesla) WakeUp() error {
	_, err := v.vehicle.Wakeup()
	return v.apiError(err)
}

var _ api.VehicleChargeController = (*Tesla)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Tesla) StartCharge() error {
	err := v.apiError(v.vehicle.StartCharging())
	if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
		return nil
	}
	return err
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Tesla) StopCharge() error {
	err := v.apiError(v.vehicle.StopCharging())

	// ignore sleeping vehicle
	if errors.Is(err, api.ErrAsleep) {
		err = nil
	}

	return err
}
