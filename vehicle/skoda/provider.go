package skoda

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Provider implements the evcc vehicle api
type Provider struct {
	chargerG func() (interface{}, error)
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.NewCached(func() (interface{}, error) {
			return api.Charger(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		return float64(res.Battery.StateOfChargeInPercent), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		if res.Plug.ConnectionState == "Connected" {
			status = api.StatusB
		}
		if res.Plug.ConnectionState == "Charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		rng = int64(res.Battery.CruisingRangeElectricInMeters) / 1e3
	}

	return rng, err
}

// var _ api.VehicleStartCharge = (*Provider)(nil)

// // StartCharge implements the api.VehicleStartCharge interface
// func (v *Provider) StartCharge() error {
// 	return v.action(ActionCharge, ActionChargeStart)
// }

// var _ api.VehicleStopCharge = (*Provider)(nil)

// // StopCharge implements the api.VehicleStopCharge interface
// func (v *Provider) StopCharge() error {
// 	return v.action(ActionCharge, ActionChargeStop)
// }
