package ford

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/vehicle/ford/autonomic"
)

type Provider struct {
	statusG  func() (autonomic.MetricsResponse, error)
	refreshG func() error
}

func NewProvider(api *autonomic.API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (autonomic.MetricsResponse, error) {
			return api.Status(vin)
		}, cache),
		refreshG: func() error {
			_, err := api.Refresh(vin)
			return err
		},
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return res.Metrics.XevBatteryStateOfCharge.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		return int64(res.Metrics.XevBatteryRange.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusNone

	res, err := v.statusG()
	if err == nil {
		switch res.Metrics.XevPlugChargerStatus.Value {
		case "DISCONNECTED":
			status = api.StatusA // disconnected
		case "CONNECTED":
			status = api.StatusB // connected, not charging
		case "CHARGING", "CHARGINGAC":
			status = api.StatusC // charging
		default:
			err = fmt.Errorf("unknown charge status: %s", res.Metrics.XevPlugChargerStatus.Value)
		}
	}

	return status, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.Metrics.Odometer.Value, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	loc := res.Metrics.Position.Value.Location
	return loc.Lat, loc.Lon, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	return v.refreshG()
}
