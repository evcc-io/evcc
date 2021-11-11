package vehicle

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/volvo"
)

// Volvo is an api.Vehicle implementation for Volvo. cars
type Volvo struct {
	*embed
	*request.Helper
	user, password, vin string
	statusG             func() (interface{}, error)
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

	log := util.NewLogger("volvo").Redact(cc.User, cc.Password, cc.VIN)

	v := &Volvo{
		embed:    &cc.embed,
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      cc.VIN,
	}

	v.statusG = provider.NewCached(v.status, cc.Cache).InterfaceGetter()

	var err error
	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, err
}

func (v *Volvo) request(uri string) (*http.Request, error) {
	basicAuth := base64.StdEncoding.EncodeToString([]byte(v.user + ":" + v.password))

	return request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization":     fmt.Sprintf("Basic %s", basicAuth),
		"Content-Type":      "application/json",
		"X-Device-Id":       "Device",
		"X-OS-Type":         "Android",
		"X-Originator-Type": "App",
		"X-OS-Version":      "22",
	})
}

// vehicles implements returns the list of user vehicles
func (v *Volvo) vehicles() ([]string, error) {
	var vehicles []string

	req, err := v.request(fmt.Sprintf("%s/customeraccounts", volvo.ApiURI))
	if err == nil {
		var res volvo.AccountResponse
		err = v.DoJSON(req, &res)

		for _, rel := range res.VehicleRelations {
			var vehicle volvo.VehicleRelation
			if req, err := v.request(rel); err == nil {
				if err = v.DoJSON(req, &vehicle); err != nil {
					return vehicles, err
				}

				vehicles = append(vehicles, vehicle.VehicleID)
			}
		}
	}

	return vehicles, err
}

func (v *Volvo) status() (interface{}, error) {
	var res volvo.Status

	req, err := v.request(fmt.Sprintf("%s/vehicles/%s/status", volvo.ApiURI, v.vin))
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// SoC implements the api.Vehicle interface
func (v *Volvo) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(volvo.Status); err == nil && ok {
		return float64(res.HvBattery.HvBatteryLevel), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Volvo)(nil)

// Status implements the api.ChargeState interface
func (v *Volvo) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if res, ok := res.(volvo.Status); err == nil && ok {
		switch res.HvBattery.HvBatteryChargeStatusDerived {
		case "CableNotPluggedInCar":
			return api.StatusA, nil
		case "CablePluggedInCar":
			return api.StatusB, nil
		case "Charging":
			return api.StatusC, nil
		}
	}

	return api.StatusNone, err
}

var _ api.VehicleRange = (*Volvo)(nil)

// VehicleRange implements the api.VehicleRange interface
func (v *Volvo) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(volvo.Status); err == nil && ok {
		return int64(res.HvBattery.DistanceToHVBatteryEmpty), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Volvo)(nil)

// VehicleOdometer implements the api.VehicleOdometer interface
func (v *Volvo) Odometer() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(volvo.Status); err == nil && ok {
		return float64(res.Odometer), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Volvo)(nil)

// FinishTime implements the VehicleFinishTimer interface
func (v *Volvo) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if res, ok := res.(volvo.Status); err == nil && ok {
		timestamp, err := time.Parse("2006-01-02T15:04:05-0700", res.HvBattery.TimeToHVBatteryFullyChargedTimestamp)

		if err == nil {
			timestamp = timestamp.Add(time.Duration(res.HvBattery.DistanceToHVBatteryEmpty) * time.Minute)
			if timestamp.Before(time.Now()) {
				return time.Time{}, api.ErrNotAvailable
			}
		}

		return timestamp, err
	}

	return time.Time{}, err
}
