package mercedes

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	dataG   func() (StatusResponse, error)
	statusG func() (api.ChargeStatus, error)
}

func NewProvider(client *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (api.ChargeStatus, error) {
			return client.identity.chargeStatus(vin)
		}, cache),
		dataG: util.Cached(func() (StatusResponse, error) {
			return client.Status(vin)
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

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface.
func (v *Provider) Status() (api.ChargeStatus, error) {
	return v.statusG()
}
