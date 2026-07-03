package eudataact

import (
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	// portalLatency is the margin added to a dataset's timestamp before the
	// following dataset is expected to be available for download
	portalLatency = time.Minute
)

// Provider implements the vehicle api on top of the EU Data Act dataset.
//
// The portal is not a live api: it stores a new dataset roughly every
// portalInterval and only ever appends. Rather than re-reading the full history
// on every poll, a store downloads each dataset once and merges its data points
// into a single map, keeping the newest value per field across all datasets. The
// status getter is cached; instead of relying on the cache ttl alone, each read
// schedules a reset for the moment the next dataset is expected (the dataset's
// timestamp plus portalInterval and a latency margin), so the map is updated as
// soon as the portal delivers a new dataset.
type Provider struct {
	statusG func() ([]point, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(log *util.Logger, api *API, vin string, cache time.Duration) *Provider {
	v := &Provider{}
	s := sharedStore(api)

	var cached util.Cacheable[[]point]
	cached = util.ResettableCached(func() ([]point, error) {
		ts, err := s.update(vin)
		if err != nil {
			log.ERROR.Println(err)
		} else if !ts.IsZero() {
			time.AfterFunc(resetDelay(ts, cache), cached.Reset)
		}
		return s.snapshot(vin), nil
	}, cache)

	v.statusG = cached.Get

	return v
}

// resetDelay returns the delay until the dataset following the one delivered at
// ts is expected to be available. It never returns less than portalLatency so a
// late or repeated dataset does not cause immediate re-polling.
func resetDelay(ts time.Time, cache time.Duration) time.Duration {
	if d := time.Until(ts.Add(cache + portalLatency)); d > portalLatency {
		return d
	}
	return portalLatency
}

// lookup returns the freshest present value among the given field names (most to
// least authoritative); the highest Seq wins, equal Seq keeps the priority order.
func lookup(data []point, fields ...string) *point {
	var best *point
	for _, f := range fields {
		if p := find(data, f); p != nil && (best == nil || p.Seq > best.Seq) {
			best = p
		}
	}
	return best
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	data, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if p := lookup(data, FieldHvBatteryLevelValue, FieldSoc, FieldHvSoc, KeyBatteryStateReportSoc); p != nil {
		return strconv.ParseFloat(p.Value, 64)
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	data, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if p := lookup(data, FieldRangeSecondary, FieldRangePrimary, FieldRangeCombined, KeyRangeID3); p != nil {
		f, err := strconv.ParseFloat(p.Value, 64)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	data, err := v.statusG()
	if err != nil {
		return time.Time{}, err
	}

	if p := lookup(data, FieldRemainingTime); p != nil && p.Value != "65535" {
		if v, err := strconv.ParseInt(p.Value, 0, 64); err == nil {
			return p.Timestamp.Add(time.Duration(v) * time.Minute), nil
		}
	}

	return time.Time{}, api.ErrNotAvailable
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	data, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if p := lookup(data, FieldOdometer, FieldOdometerValue); p != nil {
		return strconv.ParseFloat(p.Value, 64)
	}

	return 0, api.ErrNotAvailable
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	data, err := v.statusG()
	if err != nil {
		return status, err
	}

	// block 1: explicit plug state
	if p := lookup(data, FieldPlugState, FieldChargingPlug1ConnectionState); p != nil && strings.EqualFold(p.Value, "connected") {
		status = api.StatusB
	}

	// block 2: flat charging_state field and the current_charge_state field
	if p := lookup(data, FieldChargingState, FieldCurrentChargeState); p != nil &&
		(strings.EqualFold(p.Value, "charging") || strings.Contains(strings.ToUpper(p.Value), "CHARGING_HV") ||
			strings.EqualFold(p.Value, "conservationCharging") || strings.EqualFold(p.Value, "CHARGE_STATE_CONSERVATION_CHARGING")) {
		status = api.StatusC
	}

	// block 3: charging_scenario is the most explicit plug/charge signal
	if p := lookup(data, FieldChargingScenario); p != nil && status != api.StatusC {
		upper := strings.ToUpper(p.Value)
		switch {
		case strings.HasSuffix(upper, "_ACTIVE"):
			status = api.StatusC
		// the car reports finished if it's plugged in but not charging
		case strings.HasSuffix(upper, "_FINISHED"):
			status = api.StatusB
		}
	}

	return status, nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	data, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if p := lookup(data, FieldTargetSoc); p != nil {
		f, err := strconv.ParseFloat(p.Value, 64)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}
