package eudataact

import (
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	// portalInterval is the cadence at which the portal delivers a new dataset
	portalInterval = 15 * time.Minute
	// portalLatency is the margin added to a dataset's timestamp before the
	// following dataset is expected to be available for download
	portalLatency = 30 * time.Second
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
	statusG func() (map[string]point, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	v := &Provider{}
	s := newStore(api, vin)

	var cached util.Cacheable[map[string]point]
	cached = util.ResettableCached(func() (map[string]point, error) {
		ts, err := s.update()
		if err != nil {
			return nil, err
		}
		if !ts.IsZero() {
			time.AfterFunc(resetDelay(ts, time.Now()), cached.Reset)
		}
		return s.snapshot(), nil
	}, cache)

	v.statusG = cached.Get

	return v
}

// resetDelay returns the delay until the dataset following the one delivered at
// ts is expected to be available. It never returns less than portalLatency so a
// late or repeated dataset does not cause immediate re-polling.
func resetDelay(ts, now time.Time) time.Duration {
	if d := ts.Add(portalInterval + portalLatency).Sub(now); d > portalLatency {
		return d
	}
	return portalLatency
}

// lookup returns the first present, non-empty value among the given field names
func lookup(data map[string]point, fields ...string) (string, bool) {
	for _, f := range fields {
		if v, ok := data[f]; ok && v.Value != "" {
			return v.Value, true
		}
	}
	return "", false
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	data, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if s, ok := lookup(data, FieldSoc, FieldHvSoc); ok {
		return strconv.ParseFloat(s, 64)
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

	if s, ok := lookup(data, FieldRange, FieldRangePrimary); ok {
		f, err := strconv.ParseFloat(s, 64)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	data, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if s, ok := lookup(data, FieldOdometer); ok {
		return strconv.ParseFloat(s, 64)
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

	if s, ok := lookup(data, FieldPlugState); ok && strings.EqualFold(s, "connected") {
		status = api.StatusB
	}

	if s, ok := lookup(data, FieldChargingState); ok && strings.EqualFold(s, "charging") {
		status = api.StatusC
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

	if s, ok := lookup(data, FieldTargetSoc); ok {
		f, err := strconv.ParseFloat(s, 64)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}
