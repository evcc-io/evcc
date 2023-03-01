package renault

import (
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/renault/kamereon"
	"golang.org/x/exp/slices"
)

// Provider is an api.Vehicle implementation for PSA cars
type Provider struct {
	batteryG func() (kamereon.Response, error)
	cockpitG func() (kamereon.Response, error)
	hvacG    func() (kamereon.Response, error)
	wakeup   func() (kamereon.Response, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *kamereon.API, accountID, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		batteryG: provider.Cached(func() (kamereon.Response, error) {
			return api.Battery(accountID, vin)
		}, cache),
		cockpitG: provider.Cached(func() (kamereon.Response, error) {
			return api.Cockpit(accountID, vin)
		}, cache),
		hvacG: provider.Cached(func() (kamereon.Response, error) {
			return api.Hvac(accountID, vin)
		}, cache),
		wakeup: func() (kamereon.Response, error) {
			return api.WakeUp(accountID, vin)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.batteryG()

	if err == nil {
		return float64(res.Data.Attributes.BatteryLevel), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.batteryG()
	if err == nil {
		if res.Data.Attributes.PlugStatus > 0 {
			status = api.StatusB
		}
		if res.Data.Attributes.ChargingStatus >= 1.0 {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.batteryG()

	if err == nil {
		return int64(res.Data.Attributes.BatteryAutonomy), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.cockpitG()

	if err == nil {
		return res.Data.Attributes.TotalMileage, nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.batteryG()

	if err == nil {
		timestamp, err := time.Parse(time.RFC3339, res.Data.Attributes.Timestamp)

		if res.Data.Attributes.RemainingTime == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return timestamp.Add(time.Duration(*res.Data.Attributes.RemainingTime) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.hvacG()

	// Zoe Ph2, Megane e-tech
	if err, ok := err.(request.StatusError); ok && err.HasStatus(http.StatusForbidden, http.StatusBadGateway) {
		return false, api.ErrNotAvailable
	}

	if err == nil {
		state := strings.ToLower(res.Data.Attributes.HvacStatus)
		if state == "" {
			return false, api.ErrNotAvailable
		}

		active := !slices.Contains([]string{"off", "false", "invalid", "error"}, state)
		return active, nil
	}

	return false, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	_, err := v.wakeup()
	return err
}
