package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/volvo"
)

// Volvo is an api.Vehicle implementation for Volvo. cars
type Volvo struct {
	*embed
	*request.Helper
	vin     string
	statusG func() (volvo.Status, error)
}

func init() {
	registry.Add("volvo", NewVolvoFromConfig)
}

// NewVolvoFromConfig creates a new vehicle
func NewVolvoFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	basicAuth := transport.BasicAuthHeader(cc.User, cc.Password)

	log := util.NewLogger("volvo").Redact(cc.User, cc.Password, cc.VIN, basicAuth)

	v := &Volvo{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
		vin:    cc.VIN,
	}

	v.Client.Transport = &transport.Decorator{
		Base: v.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization":     basicAuth,
			"Content-Type":      "application/json",
			"X-Device-Id":       "Device",
			"X-OS-Type":         "Android",
			"X-Originator-Type": "App",
			"X-OS-Version":      "22",
		}),
	}

	v.statusG = provider.Cached(v.StatusRequest, cc.Cache)

	var err error
	v.vin, err = ensureVehicle(cc.VIN, v.Vehicles)

	return v, err
}

// vehicles implements returns the list of user vehicles
func (v *Volvo) Vehicles() ([]string, error) {
	var vehicles []string

	uri := fmt.Sprintf("%s/customeraccounts", volvo.ApiURI)

	var res volvo.AccountResponse
	err := v.GetJSON(uri, &res)
	if err == nil {
		for _, rel := range res.VehicleRelations {
			var vehicle volvo.VehicleRelation
			if err := v.GetJSON(rel, &vehicle); err != nil {
				return vehicles, err
			}

			vehicles = append(vehicles, vehicle.VehicleID)
		}
	} else if res.ErrorLabel != "" {
		err = fmt.Errorf("%w: %s: %s", err, res.ErrorLabel, res.ErrorDescription)
	}

	return vehicles, err
}

func (v *Volvo) StatusRequest() (volvo.Status, error) {
	var res volvo.Status

	uri := fmt.Sprintf("%s/vehicles/%s/status", volvo.ApiURI, v.vin)
	err := v.GetJSON(uri, &res)
	if err != nil && res.ErrorLabel != "" {
		err = fmt.Errorf("%w: %s: %s", err, res.ErrorLabel, res.ErrorDescription)
	}

	return res, err
}

// Soc implements the api.Vehicle interface
func (v *Volvo) Soc() (float64, error) {
	res, err := v.statusG()
	return float64(res.HvBattery.HvBatteryLevel), err
}

var _ api.ChargeState = (*Volvo)(nil)

// Status implements the api.ChargeState interface
func (v *Volvo) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err == nil {
		switch res.HvBattery.HvBatteryChargeStatusDerived {
		case "CableNotPluggedInCar":
			return api.StatusA, nil
		case "CablePluggedInCar", "CablePluggedInCar_FullyCharged", "CablePluggedInCar_ChargingPaused":
			return api.StatusB, nil
		case "Charging", "CablePluggedInCar_Charging":
			return api.StatusC, nil
		}
	}

	return api.StatusNone, err
}

var _ api.VehicleRange = (*Volvo)(nil)

// VehicleRange implements the api.VehicleRange interface
func (v *Volvo) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.HvBattery.DistanceToHVBatteryEmpty), err
}

var _ api.VehicleOdometer = (*Volvo)(nil)

// VehicleOdometer implements the api.VehicleOdometer interface
func (v *Volvo) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.Odometer / 1e3, err
}

var _ api.VehicleFinishTimer = (*Volvo)(nil)

// FinishTime implements the VehicleFinishTimer interface
func (v *Volvo) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err == nil {
		timestamp := res.HvBattery.TimeToHVBatteryFullyChargedTimestamp.Add(time.Duration(res.HvBattery.DistanceToHVBatteryEmpty) * time.Minute)
		if timestamp.Before(time.Now()) {
			return time.Time{}, api.ErrNotAvailable
		}

		return timestamp, err
	}

	return time.Time{}, err
}
