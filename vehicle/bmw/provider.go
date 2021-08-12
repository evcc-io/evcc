package bmw

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
			return api.Dynamic(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(DynamicResponse); err == nil && ok {
		return res.AttributesMap.ChargingLevelHv, nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if res, ok := res.(DynamicResponse); err == nil && ok {
		if res.AttributesMap.ConnectorStatus == "CONNECTED" {
			status = api.StatusB
		}
		if res.AttributesMap.ChargingHVStatus == "CHARGING" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.chargerG()
	if res, ok := res.(DynamicResponse); err == nil && ok {
		ctr := res.AttributesMap.ChargingTimeRemaining
		return time.Now().Add(time.Duration(ctr) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	if res, ok := res.(DynamicResponse); err == nil && ok {
		rng = int64(res.AttributesMap.BERemainingRangeElectricKm)
	}

	return rng, err
}
