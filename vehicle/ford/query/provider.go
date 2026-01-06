package query

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	telemetryG func() (Telemetry, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		telemetryG: util.Cached(func() (Telemetry, error) {
			return api.Telemetry(vin)
		}, cache),
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.telemetryG()
	return res.Metrics.XevBatteryStateOfCharge.Value, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.telemetryG()
	return int64(res.Metrics.XevBatteryRange.Value), err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA

	res, err := v.telemetryG()
	if err != nil {
		return status, err
	}

	switch res.Metrics.XevPlugChargerStatus.Value {
	case "CONNECTED":
		status = api.StatusB
	case "CHARGING":
		status = api.StatusC
	}

	return status, nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.telemetryG()
	return res.Metrics.Odometer.Value, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.telemetryG()
	return res.Metrics.Position.Value.Location.Lat, res.Metrics.Position.Value.Location.Lon, err
}
