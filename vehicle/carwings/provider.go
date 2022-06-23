package carwings

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Provider struct {
	chargerG  func() (ChargerResponse, error)
	climaterG func() (ClimaterResponse, error)
}

func NewProvider(api *API, vid string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.Cached(func() (ChargerResponse, error) {
			return api.Charger(vid)
		}, cache),
		climaterG: provider.Cached(func() (ClimaterResponse, error) {
			return api.Climater(vid)
		}, cache),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if err == nil {
		return float64(res.BatteryStatusRecord.BatteryStatus.SOC.Value), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.chargerG()
	if err == nil {
		vehicleRange, err := res.BatteryStatusRecord.CruisingRangeAcOn.Int64()
		if err != nil {
			return 0, err
		}
		return vehicleRange / 1000, nil
	}
	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if err == nil {
		if res.BatteryStatusRecord.PluginState == "CONNECTED" {
			status = api.StatusB // connected, not charging
		}
		if res.BatteryStatusRecord.PluginState == "NORMAL_CHARGING" {
			status = api.StatusC // charging
		}
	}

	return status, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.climaterG()

	if err == nil {
		active = res.RemoteACRecords.RemoteACOperation == "START"
		targetTemp = float64(res.RemoteACRecords.PreAC_temp)
		outsideTemp = targetTemp

		return active, outsideTemp, targetTemp, err
	}

	return false, 0, 0, err
}
