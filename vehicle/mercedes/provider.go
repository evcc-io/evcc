package mercedes

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	dataG func() (StatusResponse, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		dataG: util.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.dataG()
	return res.EvInfo.Battery.StateOfCharge, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.dataG()
	return int64(res.EvInfo.Battery.DistanceToEmpty.Value), err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.dataG()
	return res.Preconditioning.Active, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.dataG()
	return res.LocationResponse.Latitude, res.LocationResponse.Longitude, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	data, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now()
	res := time.Date(now.Year(), now.Month(), now.Day(), 0, data.EvInfo.Battery.EndOfChargeTime, 0, 0, now.Location())

	if res.Before(now) {
		res = res.Add(24 * time.Hour)
	}
	return res, nil
}
