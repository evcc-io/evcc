package mercedes

import (
	"time"

	"github.com/evcc-io/evcc/provider"
)

// Provider implements the vehicle api
type Provider struct {
	chargerG func() (EVResponse, error)
	rangeG   func() (EVResponse, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.Cached(func() (EVResponse, error) {
			return api.Soc(vin)
		}, cache),
		rangeG: provider.Cached(func() (EVResponse, error) {
			return api.Range(vin)
		}, cache),
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.chargerG()
	if err == nil {
		return float64(res.Soc.Value), nil
	}

	return 0, err
}

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.rangeG()
	if err == nil {
		return int64(res.RangeElectric.Value), nil
	}

	return 0, err
}
