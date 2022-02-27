package mercedes

import (
	"time"

	"github.com/evcc-io/evcc/provider"
)

// Provider implements the evcc vehicle api
type Provider struct {
	chargerG func() (interface{}, error)
	rangeG   func() (interface{}, error)
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.NewCached(func() (interface{}, error) {
			return api.SoC(vin)
		}, cache).InterfaceGetter(),
		rangeG: provider.NewCached(func() (interface{}, error) {
			return api.Range(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(EVResponse); err == nil && ok {
		return float64(res.SoC.Value), nil
	}

	return 0, err
}

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	if res, ok := res.(EVResponse); err == nil && ok {
		return int64(res.RangeElectric.Value), nil
	}

	return 0, err
}
