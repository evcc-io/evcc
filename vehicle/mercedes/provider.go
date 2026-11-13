package mercedes

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	protos "github.com/evcc-io/evcc/vehicle/mercedes/pb"
)

const kmPerMile = 1.609344

type Provider struct {
	dataG func() (StatusResponse, error)
	pushG func() (*protos.VEPUpdate, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		dataG: util.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
		pushG: util.Cached(func() (*protos.VEPUpdate, error) {
			return api.Push(vin)
		}, cache),
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.dataG()
	return res.EvInfo.Battery.StateOfCharge, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.dataG()
	return int64(res.EvInfo.Battery.DistanceToEmpty.Value), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.pushG()
	if err != nil {
		return 0, err
	}

	odo, ok := res.GetAttributes()["odo"]
	if !ok || odo.GetNilValue() {
		return 0, api.ErrNotAvailable
	}

	km := float64(odo.GetIntValue())
	if odo.GetDistanceUnit() == protos.VehicleAttributeStatus_MILES {
		km *= kmPerMile
	}

	return km, nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.pushG()
	if err != nil {
		return 0, err
	}

	limit, ok := res.GetAttributes()["maxSoc"]
	if !ok || limit.GetNilValue() {
		return 0, api.ErrNotAvailable
	}

	return limit.GetIntValue(), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.pushG()
	if err != nil {
		return api.StatusNone, err
	}

	// charging variants map to C, plugged without energy flow to B, unplugged (3) and unknown to A
	if status, ok := res.GetAttributes()["chargingstatus"]; ok && !status.GetNilValue() {
		switch status.GetIntValue() {
		case 0, 5, 6, 9, 10, 11, 13, 14:
			return api.StatusC, nil
		case 1, 2, 4, 7, 8, 12, 15:
			return api.StatusB, nil
		}
	}

	return api.StatusA, nil
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.pushG()
	if err != nil {
		return false, err
	}

	// preconditioning is reported by two attributes, either one set means active
	attrs := res.GetAttributes()
	return attrs["precondNow"].GetBoolValue() || attrs["precondActive"].GetBoolValue(), nil
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.dataG()
	return res.LocationResponse.Latitude, res.LocationResponse.Longitude, err
}
