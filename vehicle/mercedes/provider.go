package mercedes

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	pb "github.com/evcc-io/evcc/vehicle/mercedes/pb"
)

// Provider reads vehicle data from two sources, both cached with the same TTL:
//   - dataG: the REST widget – soc, range, preconditioning, end-of-charge time.
//   - vsuG:  the VSU push websocket – odometer, charging status, SoC limit and
//     position, which Mercedes removed from the REST widget.
//
// Each method deliberately reads from exactly one source; see the per-method
// comments for why.
type Provider struct {
	dataG func() (StatusResponse, error)
	vsuG  func() (StatusResponse, error)
}

func NewProvider(api *API, ws *Websocket, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		dataG: util.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
		vsuG: util.Cached(func() (StatusResponse, error) {
			return vsuStatus(ws, vin, cache)
		}, cache),
	}
	return impl
}

// vsuStatus reads the latest VehicleStatusUpdate from the websocket cache and
// maps it. It applies a freshness guard: if the cached update is older than the
// staleness window (based on cache, min 15m) it is treated as stale and an error
// is returned so callers retry rather than record outdated data.
func vsuStatus(ws *Websocket, vin string, cache time.Duration) (StatusResponse, error) {
	vsu, updated, ok := ws.Status(vin)
	if !ok {
		// No full update received yet – retry semantics.
		return StatusResponse{}, api.ErrMustRetry
	}

	stale := max(15*time.Minute, 3*cache)
	if time.Since(updated) > stale {
		return StatusResponse{}, errors.New("vsu data is stale")
	}

	return mapVSU(vsu), nil
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

// Odometer implements the api.VehicleOdometer interface. The odometer is only
// available via the VSU push stream (Mercedes removed it from the REST widget).
func (v *Provider) Odometer() (float64, error) {
	res, err := v.vsuG()
	return float64(res.VehicleInfo.Odometer.Value), err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface. Sourced from the VSU push
// stream since Mercedes removed chargingstatus from the REST widget.
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.vsuG()
	if err != nil {
		return api.StatusA, err
	}
	return MapChargeStatus(res.EvInfo.Battery.ChargingStatus), nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface. Sourced from the VSU push
// stream since Mercedes removed maxSoc from the REST widget.
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.vsuG()
	return int64(res.EvInfo.Battery.SocLimit), err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.dataG()
	return res.Preconditioning.Active, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.vsuG()
	return res.LocationResponse.Latitude, res.LocationResponse.Longitude, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	data, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now()
	res := time.Date(now.Year(), now.Month(), now.Day(), 0, data.EvInfo.Battery.EndOfChargeTime, 0, 0, now.Location())

	if res.Before(now) {
		res = res.Add(24 * time.Hour)
	}
	return res, nil
}

// MapChargeStatus maps the Mercedes charging status enum to an evcc charge
// status. Switching on the generated enum (rather than raw ints) keeps this
// table checked against the proto definition.
func MapChargeStatus(status pb.Chargingstatus) api.ChargeStatus {
	switch status {
	case pb.Chargingstatus_CHARGINGSTATUS_CHARGING,
		pb.Chargingstatus_CHARGINGSTATUS_SLOW_CHARGING,
		pb.Chargingstatus_CHARGINGSTATUS_FAST_CHARGING,
		pb.Chargingstatus_CHARGINGSTATUS_SLOW_CHARGING_AFTER_REACHING_TRIP_TARGET,
		pb.Chargingstatus_CHARGINGSTATUS_CHARGING_AFTER_REACHING_TRIP_TARGET,
		pb.Chargingstatus_CHARGINGSTATUS_FAST_CHARGING_AFTER_REACHING_TRIP_TARGET,
		pb.Chargingstatus_CHARGINGSTATUS_AC_CHARGING_ACTIVE,
		pb.Chargingstatus_CHARGINGSTATUS_DC_CHARGING_ACTIVE:
		// actively charging
		return api.StatusC
	case pb.Chargingstatus_CHARGINGSTATUS_END_OF_CHARGE,
		pb.Chargingstatus_CHARGINGSTATUS_CHARGE_BREAK,
		pb.Chargingstatus_CHARGINGSTATUS_CHARGING_ERROR,
		pb.Chargingstatus_CHARGINGSTATUS_DISCHARGING,
		pb.Chargingstatus_CHARGINGSTATUS_NO_CHARGING,
		pb.Chargingstatus_CHARGINGSTATUS_COMMUNICATION_WITH_EVSE_ACTIVE_NO_ENERGY_FLOW,
		pb.Chargingstatus_CHARGINGSTATUS_SOH_BATTERY_CALIBRATION_ACTIVE:
		// connected but not charging
		return api.StatusB
	}
	// CHARGE_CABLE_UNPLUGGED, UNKNOWN: disconnected
	return api.StatusA
}
