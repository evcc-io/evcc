package porsche

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the evcc vehicle interfaces on top of the PPA API.
type Provider struct {
	api     *API
	mu      sync.Mutex
	vin     string
	statusG func() (StatusResponse, error)
}

// NewProvider creates a Porsche Connect vehicle data provider. If vin is empty
// it is resolved lazily from the account's vehicle list on first use (the first
// vehicle is used), so the user does not have to enter it manually.
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	v := &Provider{
		api: api,
		vin: vin,
	}
	v.statusG = util.Cached(v.status, cache)
	return v
}

func (v *Provider) status() (StatusResponse, error) {
	vin, err := v.resolveVIN()
	if err != nil {
		return StatusResponse{}, err
	}
	return v.api.Status(vin)
}

// resolveVIN returns the configured VIN. If none was configured it auto-detects
// the vehicle, but only when the account holds exactly one - with multiple
// vehicles the VIN is ambiguous, so the user must set it explicitly (the error
// lists the available VINs). The result is cached for subsequent calls.
func (v *Provider) resolveVIN() (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.vin != "" {
		return v.vin, nil
	}

	vehicles, err := v.api.Vehicles()
	if err != nil {
		return "", err
	}

	switch len(vehicles) {
	case 0:
		return "", errors.New("no vehicles found on Porsche account")
	case 1:
		v.vin = vehicles[0].VIN
		return v.vin, nil
	default:
		list := make([]string, len(vehicles))
		for i, veh := range vehicles {
			if veh.ModelName != "" {
				list[i] = fmt.Sprintf("%s (%s)", veh.VIN, veh.ModelName)
			} else {
				list[i] = veh.VIN
			}
		}
		return "", fmt.Errorf("multiple vehicles found, set vin to one of: %s", strings.Join(list, ", "))
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface (high-voltage battery level).
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	var bl batteryLevel
	if !res.decode("BATTERY_LEVEL", &bl) {
		return 0, api.ErrNotAvailable
	}
	return bl.Percent, nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface.
//
// The PPA API has no explicit "plugged" flag; we derive the EVSE status from
// CHARGING_SUMMARY.status and the live charging power. Observed status values:
// "NOT_PLUGGED" (disconnected), "CHARGING_COMPLETED" (plugged, done), "CHARGING"
// (charging). We treat any "not plugged"-style status as disconnected and any
// remaining plugged state (completed/paused/error) as connected.
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	var rate chargingRate
	res.decode("CHARGING_RATE", &rate)

	var summary chargingSummary
	if !res.decode("CHARGING_SUMMARY", &summary) {
		return api.StatusNone, api.ErrNotAvailable
	}

	status := strings.ToUpper(summary.Status)

	switch {
	case status == "CHARGING" || rate.ChargingPower > 0:
		return api.StatusC, nil
	case strings.Contains(status, "NOT_PLUGGED"),
		strings.Contains(status, "UNPLUGGED"),
		strings.Contains(status, "DISCONNECTED"):
		return api.StatusA, nil
	default:
		// plugged but not charging (e.g. CHARGING_COMPLETED, NOT_CHARGING, ERROR)
		return api.StatusB, nil
	}
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface (electric range).
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	var r rangeValue
	if !res.decode("E_RANGE", &r) {
		return 0, api.ErrNotAvailable
	}
	return int64(r.Kilometers), nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface.
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	var m rangeValue
	if !res.decode("MILEAGE", &m) {
		return 0, api.ErrNotAvailable
	}
	return m.Kilometers, nil
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface.
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	if err != nil {
		return false, err
	}

	var c climatizerState
	if !res.decode("CLIMATIZER_STATE", &c) {
		return false, api.ErrNotAvailable
	}
	return c.IsOn, nil
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface.
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, 0, err
	}

	var g gpsLocation
	if !res.decode("GPS_LOCATION", &g) {
		return 0, 0, api.ErrNotAvailable
	}

	lat, lng, ok := strings.Cut(g.Location, ",")
	if !ok {
		return 0, 0, api.ErrNotAvailable
	}

	latF, err := strconv.ParseFloat(strings.TrimSpace(lat), 64)
	if err != nil {
		return 0, 0, err
	}
	lngF, err := strconv.ParseFloat(strings.TrimSpace(lng), 64)
	if err != nil {
		return 0, 0, err
	}
	return latF, lngF, nil
}
