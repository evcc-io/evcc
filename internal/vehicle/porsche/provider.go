package porsche

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

type vehicleResponse struct {
	CarControlData struct {
		BatteryLevel struct {
			Unit  string
			Value float64
		}
		Mileage struct {
			Unit  string
			Value float64
		}
		RemainingRanges struct {
			ElectricalRange struct {
				Distance struct {
					Unit  string
					Value float64
				}
			}
		}
	}
}

// Provider is an api.Vehicle implementation for PSA cars
type Provider struct {
	api     *API
	statusG func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		api: api,
	}

	impl.statusG = provider.NewCached(func() (interface{}, error) {
		return impl.status(vin)
	}, cache).InterfaceGetter()

	return impl
}

// Status implements the vehicle status repsonse
func (v *Provider) status(vin string) (interface{}, error) {
	uri := fmt.Sprintf("https://connect-portal.porsche.com/core/api/v3/de/de_DE/vehicles/%s", vin)
	req, err := v.api.request(uri, false)
	if err != nil {
		return 0, err
	}

	var pr vehicleResponse
	err = v.api.DoJSON(req, &pr)

	return pr, err
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(vehicleResponse); err == nil && ok {
		return res.CarControlData.BatteryLevel.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(vehicleResponse); err == nil && ok {
		return int64(res.CarControlData.RemainingRanges.ElectricalRange.Distance.Value), nil
	}

	return 0, err
}
