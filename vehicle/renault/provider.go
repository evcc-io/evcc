package renault

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/renault/kamereon"
)

// Provider is an api.Vehicle implementation for PSA cars
type Provider struct {
	batteryStatusG func() (kamereon.BatteryStatus, error)
	cockpitG       func() (kamereon.Cockpit, error)
	socLevelsG     func() (kamereon.SocLevels, error)
	hvacG          func() (kamereon.HvacStatus, error)
	wakeup         func() error
	position       func() (kamereon.Position, error)
	chargeAction   func(action string) (kamereon.ChargeAction, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *kamereon.API, accountID, vin string, wakeupMode string, cache time.Duration) *Provider {
	impl := &Provider{
		batteryStatusG: util.Cached(func() (kamereon.BatteryStatus, error) {
			return api.BatteryStatus(accountID, vin)
		}, cache),
		cockpitG: util.Cached(func() (kamereon.Cockpit, error) {
			return api.Cockpit(accountID, vin)
		}, cache),
		socLevelsG: util.Cached(func() (kamereon.SocLevels, error) {
			return api.SocLevels(accountID, vin)
		}, cache),
		hvacG: util.Cached(func() (kamereon.HvacStatus, error) {
			return api.HvacStatus(accountID, vin)
		}, cache),
		wakeup: func() error {
			var err error
			switch wakeupMode {
			case "alternative":
				_, err = api.ChargeAction(accountID, kamereon.ActionStart, vin)
			case "MY24":
				_, err = api.WakeUpMy24(accountID, vin)
			default:
				_, err = api.WakeUp(accountID, vin)

				// Check if default wakeup is unsupported
				var se *request.StatusError
				if errors.As(err, &se) && se.HasStatus(http.StatusForbidden, http.StatusNotFound, http.StatusBadGateway) {
					_, err = api.WakeUpMy24(accountID, vin)
				}
			}
			return err
		},
		position: func() (kamereon.Position, error) {
			return api.Position(accountID, vin)
		},
		chargeAction: func(action string) (kamereon.ChargeAction, error) {
			return api.ChargeAction(accountID, action, vin)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.batteryStatusG()
	if err != nil {
		return 0, err
	}

	if res.BatteryLevel == nil {
		return 0, api.ErrNotAvailable
	}

	return float64(*res.BatteryLevel), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.batteryStatusG()
	if err == nil {
		if res.PlugStatus == 1 {
			status = api.StatusB
		}
		if res.ChargingStatus >= 1.0 {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.batteryStatusG()

	if err == nil {
		return int64(res.BatteryAutonomy), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.cockpitG()
	if err != nil {
		return 0, err
	}

	if res.TotalMileage != nil {
		return *res.TotalMileage, nil
	}

	return 0, api.ErrNotAvailable
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.socLevelsG()

	// Check if endpoint is unavailable
	var se *request.StatusError
	if errors.As(err, &se) && se.HasStatus(http.StatusForbidden, http.StatusNotFound, http.StatusBadGateway) {
		return 0, api.ErrNotAvailable
	}

	if err != nil {
		return 0, err
	}

	if res.SocTarget != nil {
		return int64(*res.SocTarget), nil
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.batteryStatusG()

	if err == nil {
		timestamp, err := time.Parse(time.RFC3339, res.Timestamp)

		if res.RemainingTime == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return timestamp.Add(time.Duration(*res.RemainingTime) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.hvacG()

	// Zoe Ph2, Megane e-tech
	var se *request.StatusError
	if errors.As(err, &se) && se.HasStatus(http.StatusForbidden, http.StatusNotFound, http.StatusBadGateway) {
		return false, api.ErrNotAvailable
	}

	if err == nil {
		state := strings.ToLower(res.HvacStatus)
		if state == "" {
			return false, api.ErrNotAvailable
		}

		active := !slices.Contains([]string{"off", "false", "invalid", "error", "unavailable"}, state)
		return active, nil
	}

	return false, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	return v.wakeup()
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.position()
	if err == nil {
		return res.Latitude, res.Longitude, nil
	}

	return 0, 0, err
}

var _ api.ChargeController = (*Provider)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *Provider) ChargeEnable(enable bool) error {
	action := map[bool]string{true: kamereon.ActionStart, false: kamereon.ActionStop}
	_, err := v.chargeAction(action[enable])
	return err
}
