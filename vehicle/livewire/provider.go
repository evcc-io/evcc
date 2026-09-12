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
	status   util.Cacheable[ChargingStatus]
	position util.Cacheable[Position]
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, bikeID string, cache time.Duration) *Provider {
	return &Provider{
		status: util.ResettableCached(func() (ChargingStatus, error) {
			return api.Status(bikeID)
		}, cache),
		position: util.ResettableCached(func() (Position, error) {
			return api.Position(bikeID)
		}, cache),
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.status.Get()
	return res.BatteryPercentage, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.status.Get()
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
	res, err := v.status.Get()
	return int64(math.Round(res.Range * kmPerMile)), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.status.Get()
	return res.Odometer * kmPerMile, err
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.status.Get()
	return res.MaxLimit, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.status.Get()
	if err != nil {
		return time.Time{}, err
	}
	if !res.ChargingStatus || res.TimeToMaxLimit <= 0 {
		return time.Time{}, api.ErrNotAvailable
	}

	// timeToMaxLimit is in minutes and stays 0 for the first minutes of a charge
	return time.Now().Add(time.Duration(float64(res.TimeToMaxLimit) * float64(time.Minute))), nil
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.position.Get()
	return res.Latitude, res.Longitude, err
}
