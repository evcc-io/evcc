package mercedes

import (
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	dataG  func() (StatusResponse, error)
	stateG func() (*VehicleStatusUpdate, error)
}

func NewProvider(client *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		stateG: util.Cached(func() (*VehicleStatusUpdate, error) {
			return client.identity.vehicleStatus(vin)
		}, cache),
		dataG: util.Cached(func() (StatusResponse, error) {
			return client.Status(vin)
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

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface.
func (v *Provider) Status() (api.ChargeStatus, error) {
	state, err := v.stateG()
	if err != nil {
		return api.StatusNone, err
	}
	attr := state.GetChargingStatus()
	if !validMetadata(attr.GetMetadata()) {
		return api.StatusNone, api.ErrMustRetry
	}
	return mapChargeStatus(attr.GetValue())
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface.
func (v *Provider) Odometer() (float64, error) {
	state, err := v.stateG()
	if err != nil {
		return 0, err
	}
	attr := state.GetOdometer()
	if !validMetadata(attr.GetMetadata()) {
		return 0, api.ErrMustRetry
	}
	value := float64(attr.GetValue())
	if value < 0 {
		return 0, fmt.Errorf("invalid Mercedes odometer: %d", attr.GetValue())
	}
	switch attr.GetUnit() {
	case 1:
		return value, nil
	case 2:
		return value * 1.609344, nil
	default:
		return 0, fmt.Errorf("unknown Mercedes distance unit: %d", attr.GetUnit())
	}
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface.
func (v *Provider) GetLimitSoc() (int64, error) {
	state, err := v.stateG()
	if err != nil {
		return 0, err
	}
	attr := state.GetMaxSoc()
	if !validMetadata(attr.GetMetadata()) {
		return 0, api.ErrMustRetry
	}
	// Zero can mean a program-specific limit, which this subset does not decode.
	if attr.GetValue() <= 0 || attr.GetValue() > 100 {
		return 0, api.ErrNotAvailable
	}
	return attr.GetValue(), nil
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface.
func (v *Provider) Climater() (bool, error) {
	state, err := v.stateG()
	if err != nil {
		return false, err
	}
	attr := state.GetPreconditioningActive()
	if !validMetadata(attr.GetMetadata()) {
		return false, api.ErrMustRetry
	}
	return attr.GetValue(), nil
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface.
func (v *Provider) Position() (float64, float64, error) {
	state, err := v.stateG()
	if err != nil {
		return 0, 0, err
	}
	lat, lon := state.GetLatitude(), state.GetLongitude()
	if !validMetadata(lat.GetMetadata()) || !validMetadata(lon.GetMetadata()) {
		return 0, 0, api.ErrMustRetry
	}
	x, y := lat.GetValue(), lon.GetValue()
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.Abs(x) > 90 || math.Abs(y) > 180 {
		return 0, 0, fmt.Errorf("invalid Mercedes position")
	}
	return x, y, nil
}

func validMetadata(metadata *AttributeMetadata) bool {
	return metadata != nil && metadata.Status == 0
}
