package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/connectedcars"
)

// ConnectedCars is an api.Vehicle implementation for the Connected Cars platform (connectedcars.io).
type ConnectedCars struct {
	*embed
	dataG func() (connectedcars.VehicleData, error)
}

func init() {
	registry.Add("connected-cars", NewConnectedCarsFromConfig)
}

// NewConnectedCarsFromConfig creates a new vehicle
func NewConnectedCarsFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed       `mapstructure:",squash"`
		DeviceToken string
		Domain      string
		Namespace   string
		VIN         string
		Cache       time.Duration
	}{
		Domain:    "au1.connectedcars.io",
		Namespace: "vwaustralia:app",
		Cache:     interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.DeviceToken == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("connected-cars").Redact(cc.DeviceToken)

	api := connectedcars.NewAPI(log, cc.Domain, cc.Namespace, cc.DeviceToken)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v connectedcars.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)
	if err != nil {
		return nil, err
	}

	v := &ConnectedCars{
		embed: &cc.embed,
		dataG: util.Cached(func() (connectedcars.VehicleData, error) {
			return api.Data(vehicle.ID)
		}, cc.Cache),
	}

	return v, nil
}

// Soc implements the api.Vehicle interface
func (v *ConnectedCars) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	if res.ChargePercentage == nil {
		return 0, api.ErrNotAvailable
	}
	return res.ChargePercentage.Pct, nil
}

var _ api.VehicleRange = (*ConnectedCars)(nil)

// Range implements the api.VehicleRange interface
func (v *ConnectedCars) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	if res.RangeTotalKm == nil {
		return 0, api.ErrNotAvailable
	}
	return int64(res.RangeTotalKm.Km), nil
}

var _ api.VehicleOdometer = (*ConnectedCars)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *ConnectedCars) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	if res.Odometer == nil {
		return 0, api.ErrNotAvailable
	}
	return res.Odometer.Odometer, nil
}
