package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const defaultChargeControlBackoff = 5 * time.Minute

// Provider implements the vehicle api.
// Based on https://github.com/Hacksore/bluelinky.
type Provider struct {
	statusG func() (BluelinkVehicleStatusLatest, error)
	refresh func() error
	chargeS func(bool) error

	chargeMu      sync.Mutex
	chargeBackoff time.Duration
	chargeUntil   time.Time
	chargeErr     error
}

// NewProvider creates a new BlueLink API
func NewProvider(api *API, identity *Identity, vehicle Vehicle, pin string, cache time.Duration) *Provider {
	return &Provider{
		statusG: util.Cached(func() (BluelinkVehicleStatusLatest, error) {
			return api.StatusLatest(vehicle)
		}, cache),
		refresh: func() error { return api.Refresh(vehicle) },
		chargeS: func(enable bool) error {
			return api.ChargeEnable(identity, vehicle, pin, enable)
		},
		chargeBackoff: defaultChargeControlBackoff,
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.SoC()
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusNone
	res, err := v.statusG()
	if err != nil {
		return status, err
	}
	return res.Status()
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.Range()
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.Odometer()
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	if err != nil {
		return false, err
	}
	return res.Climater()
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.GetLimitSoc()
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, 0, err
	}
	return res.Position()
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	// Triggers refresh from vehicle
	return v.refresh()
}

// EnableCharge starts or stops charging. Decorated as ChargeController only when a PIN is configured.
func (v *Provider) EnableCharge(enable bool) error {
	v.chargeMu.Lock()
	defer v.chargeMu.Unlock()

	if remaining := time.Until(v.chargeUntil); remaining > 0 {
		err := v.chargeErr
		if err == nil {
			err = api.ErrMustRetry
		}
		return fmt.Errorf("charge control paused for %v: %w", remaining.Round(time.Second), err)
	}

	err := v.chargeS(enable)
	if chargeControlShouldBackoff(err) {
		backoff := v.chargeBackoff
		if backoff <= 0 {
			backoff = defaultChargeControlBackoff
		}
		v.chargeUntil = time.Now().Add(backoff)
		v.chargeErr = err
	} else if err == nil {
		v.chargeUntil = time.Time{}
		v.chargeErr = nil
	}

	return err
}

// chargeControlShouldBackoff reports whether Hyundai rejected a remote command
// in a way that occupies the vehicle command slot. Retrying on every loadpoint
// interval keeps that slot busy and blocks the official app.
func chargeControlShouldBackoff(err error) bool {
	if err == nil {
		return false
	}

	var se *request.StatusError
	if errors.As(err, &se) && se.HasStatus(http.StatusBadRequest, http.StatusForbidden) {
		return true
	}

	if u, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range u.Unwrap() {
			if chargeControlShouldBackoff(e) {
				return true
			}
		}
	}

	return false
}
