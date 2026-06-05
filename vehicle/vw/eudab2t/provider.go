package eudab2t

import (
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api on top of the B2T pull api. The data
// endpoint returns the vehicle's current data points directly, so the getter is
// simply cached for the configured interval.
type Provider struct {
	dataG func() (map[string]string, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	return &Provider{
		dataG: util.Cached(func() (map[string]string, error) {
			return api.Data(vin)
		}, cache),
	}
}

// lookup returns the first present, non-empty value among the given field names
func lookup(data map[string]string, fields ...string) (string, bool) {
	for _, f := range fields {
		if v, ok := data[f]; ok && v != "" {
			return v, true
		}
	}
	return "", false
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	data, err := v.dataG()
	if err != nil {
		return 0, err
	}

	if s, ok := lookup(data, FieldBatteryStateReportSoc, FieldSoc, FieldHvSoc, FieldHvBatteryLevel); ok {
		return strconv.ParseFloat(s, 64)
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	data, err := v.dataG()
	if err != nil {
		return 0, err
	}

	if s, ok := lookup(data, FieldRangeSecondary, FieldRangePrimary, FieldRangeCombined); ok {
		f, err := strconv.ParseFloat(s, 64)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	data, err := v.dataG()
	if err != nil {
		return 0, err
	}

	if s, ok := lookup(data, FieldOdometer, FieldOdometerValue); ok {
		return strconv.ParseFloat(s, 64)
	}

	return 0, api.ErrNotAvailable
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	data, err := v.dataG()
	if err != nil {
		return status, err
	}

	if s, ok := lookup(data, FieldPlugState); ok && strings.EqualFold(s, "connected") {
		status = api.StatusB
	}

	if s, ok := lookup(data, FieldChargingState, FieldCurrentChargeState); ok &&
		(strings.EqualFold(s, "charging") || strings.Contains(strings.ToUpper(s), "CHARGING_HV")) {
		status = api.StatusC
	}

	return status, nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	data, err := v.dataG()
	if err != nil {
		return 0, err
	}

	if s, ok := lookup(data, FieldTargetSoc); ok {
		f, err := strconv.ParseFloat(s, 64)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}
