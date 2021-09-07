package audi

import (
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// Provider implements the evcc vehicle api
type Provider struct {
	statusG func() (interface{}, error)
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(vw.StatusResponse); err == nil && ok {
		fd := res.ServiceByID(vw.ServiceOdometer).FieldByID(vw.ServiceOdometer)
		if fd != nil {
			return strconv.ParseFloat(fd.Value, 64)
		}
	}

	return 0, err
}
