package skoda

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api
type Provider struct {
	statusG   func() (StatusResponse, error)
	chargerG  func() (ChargerResponse, error)
	settingsG func() (SettingsResponse, error)
	climateG  func() (ClimaterResponse, error)
	action    func(action, value string) error
	wakeup    func() error
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
		chargerG: util.Cached(func() (ChargerResponse, error) {
			return api.Charger(vin)
		}, cache),
		climateG: util.Cached(func() (ClimaterResponse, error) {
			return api.Climater(vin)
		}, cache),
		settingsG: util.Cached(func() (SettingsResponse, error) {
			return api.Settings(vin)
		}, cache),
		action: func(action, value string) error {
			return api.Action(vin, action, value)
		},
		wakeup: func() error {
			return api.WakeUp(vin)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.chargerG()
	if err == nil {
		return float64(res.Status.Battery.StateOfChargeInPercent), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.climateG()
	if err == nil {
		if res.ChargerConnectionState == "CONNECTED" {
			status = api.StatusB
		}
	}

	resChrg, err := v.chargerG()
	if err == nil {
		if resChrg.Status.State == "CHARGING" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.chargerG()
	if err == nil {
		crg := res.Status

		// estimate not available
		if crg.State == "Error" || crg.ChargeType == "Invalid" {
			return time.Time{}, api.ErrNotAvailable
		}

		remaining := time.Duration(crg.RemainingTimeToFullyChargedInMinutes) * time.Minute
		return time.Now().Add(remaining), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	return res.Status.Battery.RemainingCruisingRangeInMeters / 1e3, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (odo float64, err error) {
	res, err := v.statusG()
	return res.MileageInKm, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.climateG()
	return slices.Contains([]string{"COOLING", "HEATING", "HEATING_AUXILIARY", "VENTILATION", "ON"}, res.State), err
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.chargerG()
	if err == nil {
		if res.Settings.TargetStateOfChargeInPercent == nil {
			return 0, api.ErrNotAvailable
		}
		return int64(*res.Settings.TargetStateOfChargeInPercent), nil
	}

	return 0, err
}

var _ api.ChargeController = (*Provider)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *Provider) ChargeEnable(enable bool) error {
	action := map[bool]string{true: ActionChargeStart, false: ActionChargeStop}
	return v.action(ActionCharge, action[enable])
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	return v.wakeup()
}
