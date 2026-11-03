package livewire

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// kmPerMile converts the backend's imperial range and odometer, confirmed
// against the app which shows 886 km for an odometer value of 550.63
const kmPerMile = 1.609344

// Provider implements the vehicle api
type Provider struct {
	status   func() (ChargingStatus, error)
	position func() (Position, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, bikeID string, cache time.Duration) *Provider {
	return &Provider{
		status: util.Cached(func() (ChargingStatus, error) {
			res, err := api.Status(bikeID)
			res.Received = time.Now()
			return res, err
		}, cache),
		position: util.Cached(func() (Position, error) {
			return api.Position(bikeID)
		}, cache),
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.status()
	return res.BatteryPercentage, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.status()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA
	if res.PluggedIn {
		status = api.StatusB
	}
	if res.ChargingStatus {
		status = api.StatusC
	}

	return status, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.status()
	return int64(math.Round(res.Range * kmPerMile)), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.status()
	return res.Odometer * kmPerMile, err
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.status()
	if err == nil && res.MaxLimit == 0 {
		err = api.ErrNotAvailable
	}
	return res.MaxLimit, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.position()
	return res.Latitude, res.Longitude, err
}
