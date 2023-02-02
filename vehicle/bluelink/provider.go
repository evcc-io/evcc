package bluelink

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

// minimal interval to wait between wakeups
const wakeupTimeout = 5 * time.Minute

// Provider implements the vehicle api.
// Based on https://github.com/Hacksore/bluelinky.
type Provider struct {
	api *API
	vid string

	statusAge   time.Duration
	cacheExpiry time.Duration

	mu              sync.Mutex
	fetchStatusTime time.Time
	forceUpdateTime time.Time

	cachedStatusValid   bool
	fetchStatusHadError bool

	cachedStatus          VehicleStatus
	cachedVehicleLocation VehicleLocation
	cachedOdometer        Odometer
}

// NewProvider creates a new BlueLink API
func NewProvider(api *API, vid string, statusAge, cacheExpiry time.Duration) *Provider {
	v := &Provider{
		api: api,
		vid: vid,

		statusAge:   statusAge,
		cacheExpiry: cacheExpiry,
	}

	return v
}

func (v *Provider) fetchServerStatus() error {
	v.fetchStatusTime = time.Now()
	serverStatus, err := v.api.StatusLatest(v.vid)
	if err != nil {
		v.fetchStatusHadError = true
	} else {
		v.fetchStatusHadError = false
		v.cachedStatusValid = true
		v.cachedStatus = serverStatus.ResMsg.VehicleStatusInfo.VehicleStatus
		v.cachedVehicleLocation = serverStatus.ResMsg.VehicleStatusInfo.VehicleLocation
		v.cachedOdometer = serverStatus.ResMsg.VehicleStatusInfo.Odometer
	}
	return err
}

func (v *Provider) forceStatusUpdate() error {
	v.forceUpdateTime = time.Now()
	serverStatus, err := v.api.StatusPartial(v.vid)
	if err == nil {
		v.cachedStatusValid = true
		v.cachedStatus = serverStatus.ResMsg
	}
	return err
}

// status wraps the two api status calls and adds status refresh
func (v *Provider) status() (VehicleStatus, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Since(v.fetchStatusTime) <= v.cacheExpiry ||
		time.Since(v.forceUpdateTime) <= v.cacheExpiry {
		// return the cached version in any case if we made an api request in the last v.cacheExpiry
		if v.fetchStatusHadError {
			return VehicleStatus{}, api.ErrMustRetry
		}
		return v.cachedStatus, nil
	}

	hasFetchedStatus := false
	for {
		// skip for first time with invalid status
		if v.cachedStatusValid {
			updated, err := v.cachedStatus.Updated()
			if err != nil {
				return VehicleStatus{}, err
			}
			if time.Since(updated) <= v.statusAge {
				// cachedStatus is 'recent'
				return v.cachedStatus, nil
			}
			if hasFetchedStatus {
				// fetched status is still old -> force status update
				break
			}
		}
		// check if status on server updated before forcing update
		err := v.fetchServerStatus()
		if err != nil {
			return VehicleStatus{}, err
		}
		hasFetchedStatus = true
	}

	err := v.forceStatusUpdate()
	if err != nil {
		return VehicleStatus{}, err
	}
	return v.cachedStatus, nil
}

// forceStatusUpdate() does not include location or odometer in the response, so it needs its own getter
func (v *Provider) locationAndOdometer() (VehicleLocation, Odometer, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// check if max(v.statusAge, v.cacheExpiry) has elapsed since the last fetchStatus
	timeSinceFetch := time.Since(v.fetchStatusTime)
	if timeSinceFetch <= v.statusAge ||
		timeSinceFetch <= v.cacheExpiry {
		if v.fetchStatusHadError {
			return v.cachedVehicleLocation, v.cachedOdometer, api.ErrMustRetry
		}
		return v.cachedVehicleLocation, v.cachedOdometer, nil
	}
	// we do not use v.cachedStatus.Updated() here,
	// as the location should use v.cachedVehicleLocation.Time
	// which might be older

	// TODO bluelink: improve api.VehiclePosition:
	// - force a location update using the 'vehicles/%s/location' endpoint
	// - parse v.cachedVehicleLocation.Time to check expiry

	// just re-fetch the status maybe it updated maybe not
	err := v.fetchServerStatus()
	return v.cachedVehicleLocation, v.cachedOdometer, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Since(v.forceUpdateTime) > wakeupTimeout {
		// forcing an update will usually make the car start charging even if the (first) resulting status still says it does not charge...
		return v.forceStatusUpdate()
	}
	// do nothing if we already forced an update in the last wakeupTimeout
	return nil
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.status()

	if err == nil {
		return res.EvStatus.BatteryStatus, nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.Battery interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.status()

	status := api.StatusNone
	if err == nil {
		status = api.StatusA
		if res.EvStatus.BatteryPlugin > 0 {
			status = api.StatusB
		}
		if res.EvStatus.BatteryCharge {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.status()

	if err == nil {
		remaining := res.EvStatus.RemainTime2.Atc.Value

		if remaining == 0 {
			return time.Time{}, api.ErrNotAvailable
		}

		ts, err := res.Updated()
		return ts.Add(time.Duration(remaining) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.status()

	if err == nil {
		if dist := res.EvStatus.DrvDistance; len(dist) == 1 {
			return int64(dist[0].RangeByFuel.EvModeRange.Value), nil
		}

		return 0, api.ErrNotAvailable
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	_, odometer, err := v.locationAndOdometer()
	return odometer.Value, err
}

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Provider) TargetSoc() (float64, error) {
	res, err := v.status()

	if err == nil {
		for _, targetSOC := range res.EvStatus.ReservChargeInfos.TargetSocList {
			if targetSOC.PlugType == plugTypeAC {
				return float64(targetSOC.TargetSocLevel), nil
			}
		}
	}

	return 0, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	loc, _, err := v.locationAndOdometer()
	return loc.Coord.Lat, loc.Coord.Lon, err
}
