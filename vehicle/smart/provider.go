package smart

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// https://github.com/TA2k/ioBroker.smart-eq

type Provider struct {
	statusG func() (StatusResponse, error)
	expiry  time.Duration
}

func NewProvider(log *util.Logger, api *API, vin string, expiry, cache time.Duration) *Provider {
	v := &Provider{
		expiry: expiry,
	}

	v.statusG = provider.Cached(func() (StatusResponse, error) {
		return v.status(
			func() (StatusResponse, error) { return api.Status(vin) },
			func() (StatusResponse, error) { return api.Refresh(vin) },
		)
	}, cache)

	return v
}

func (v *Provider) status(statusG func() (StatusResponse, error), refreshG func() (StatusResponse, error)) (StatusResponse, error) {
	res, err := statusG()

	// if err == nil && res.Status.StatusData.Soc.Ts.Time.Add(v.expiry).Before(time.Now()) {
	// 	res, err = refreshG()
	// }

	return res, err
}

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	return res.Status.StatusData.Soc.Value, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.Status.StatusData.RangeElectric.Value), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the Provider.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.Status.StatusData.Odo.Value, err
}
