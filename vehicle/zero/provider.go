package zero

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api
type Provider struct {
	status util.Cacheable[State]
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, unitId string, cache time.Duration) *Provider {
	impl := &Provider{
		status: util.ResettableCached(func() (State, error) {
			return api.Status(unitId)
		}, cache),
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.status.Get()
	return float64(res.Soc), err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.status.Get()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA
	if res.Pluggedin != 0 {
		status = api.StatusB
	}
	if res.Charging != 0 {
		status = api.StatusC
	}

	return status, nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.status.Get()
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse("20060102150405", res.DatetimeActual)
	if err != nil {
		t = time.Now()
	}

	return t.Add(time.Duration(res.Chargingtimeleft) * time.Minute), nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.status.Get()
	return res.Mileage, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.status.Get()
	return res.Latitude, res.Longitude, err
}
