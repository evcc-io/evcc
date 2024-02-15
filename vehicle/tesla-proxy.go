package vehicle

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/transport"
	vc "github.com/evcc-io/evcc/vehicle/tesla-vehicle-command"
	tesla "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

// TeslaProxy is an api.Vehicle implementation for Tesla cars
type TeslaProxy struct {
	*embed
	vehicle *tesla.Vehicle
	dataG   func() (*tesla.VehicleData, error)
}

func init() {
	registry.Add("tesla-proxy", NewTeslaProxyFromConfig)
}

// NewTeslaProxyFromConfig creates a new vehicle
func NewTeslaProxyFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		Tokens Tokens
		URI    string
		VIN    string
		Cache  time.Duration
	}{
		URI:   "https://tesla.evcc.io/api/1",
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	v := &TeslaProxy{
		embed: &cc.embed,
	}

	// authenticated http client with logging injected to the Tesla client
	log := util.NewLogger("tesla-proxy").Redact(
		cc.Tokens.Access, cc.Tokens.Refresh,
		vc.OAuth2Config.ClientID, vc.OAuth2Config.ClientSecret,
	)

	identity, err := vc.NewIdentity(log, token)
	if err != nil {
		return nil, err
	}

	hc := request.NewClient(log)
	hc.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-Auth-Token": sponsor.Token,
		}),
		Base: &oauth2.Transport{
			Source: identity,
			Base:   hc.Transport,
		},
	}

	options := []tesla.ClientOption{
		tesla.WithClient(hc),
		tesla.WithBaseURL(cc.URI),
	}

	client, err := tesla.NewClient(context.Background(), options...)
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
func (v *TeslaProxy) apiError(err error) error {
	if err != nil && err.Error() == "408 Request Timeout" {
		err = api.ErrAsleep
	}
	return err
}

// Soc implements the api.Vehicle interface
func (v *TeslaProxy) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.UsableBatteryLevel), nil
}

var _ api.ChargeState = (*TeslaProxy)(nil)

// Status implements the api.ChargeState interface
func (v *TeslaProxy) Status() (api.ChargeStatus, error) {
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

var _ api.ChargeRater = (*TeslaProxy)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *TeslaProxy) ChargedEnergy() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return res.Response.ChargeState.ChargeEnergyAdded, nil
}

var _ api.VehicleRange = (*TeslaProxy)(nil)

// Range implements the api.VehicleRange interface
func (v *TeslaProxy) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return int64(kmPerMile * res.Response.ChargeState.BatteryRange), nil
}

var _ api.VehicleOdometer = (*TeslaProxy)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *TeslaProxy) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return kmPerMile * res.Response.VehicleState.Odometer, nil
}

var _ api.VehicleFinishTimer = (*TeslaProxy)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *TeslaProxy) FinishTime() (time.Time, error) {
	res, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(time.Duration(res.Response.ChargeState.MinutesToFullCharge) * time.Minute), nil
}

// TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

var _ api.VehiclePosition = (*TeslaProxy)(nil)

// Position implements the api.VehiclePosition interface
func (v *TeslaProxy) Position() (float64, float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, 0, err
	}
	return res.Response.DriveState.Latitude, res.Response.DriveState.Longitude, nil
}

var _ api.SocLimiter = (*TeslaProxy)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *TeslaProxy) TargetSoc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.ChargeLimitSoc), nil
}

var _ api.CurrentController = (*TeslaProxy)(nil)

// MaxCurrent implements the api.CurrentLimiter interface
func (v *TeslaProxy) MaxCurrent(current int64) error {
	return v.apiError(v.vehicle.SetChargingAmps(int(current)))
}

var _ api.Resurrector = (*TeslaProxy)(nil)

// WakeUp implements the api.Resurrector interface
func (v *TeslaProxy) WakeUp() error {
	_, err := v.vehicle.Wakeup()
	return v.apiError(err)
}

var _ api.VehicleChargeController = (*TeslaProxy)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *TeslaProxy) StartCharge() error {
	err := v.apiError(v.vehicle.StartCharging())
	if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
		return nil
	}
	return err
}

// StopCharge implements the api.VehicleChargeController interface
func (v *TeslaProxy) StopCharge() error {
	err := v.apiError(v.vehicle.StopCharging())

	// ignore sleeping vehicle
	if errors.Is(err, api.ErrAsleep) {
		err = nil
	}

	return err
}
