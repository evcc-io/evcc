package psa

import (
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider is an api.Vehicle implementation for PSA cars
type Provider struct {
	mu      sync.Mutex
	vid     string
	vehicle func() (string, error)
	statusG func() (Status, error)
}

// NewProvider creates a vehicle api provider. The PSA vehicle id is resolved
// lazily on first use (via the vehicle resolver), so the vehicle can be built
// before the account is authenticated via the browser login.
func NewProvider(api *API, vehicle func() (string, error), cache time.Duration) *Provider {
	v := &Provider{
		vehicle: vehicle,
	}
	v.statusG = util.Cached(func() (Status, error) {
		vid, err := v.vehicleID()
		if err != nil {
			return Status{}, err
		}
		return api.Status(vid)
	}, cache)
	return v
}

// vehicleID resolves and caches the PSA vehicle id on first successful call.
func (v *Provider) vehicleID() (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.vid == "" {
		vid, err := v.vehicle()
		if err != nil {
			return "", err
		}
		v.vid = vid
	}
	return v.vid, nil
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		for _, e := range res.Energy {
			if e.Type != "Electric" {
				continue
			}

			return e.Level, nil
		}

		err = api.ErrNotAvailable
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		for _, e := range res.Energy {
			if e.Type != "Electric" {
				continue
			}

			return int64(e.Autonomy), nil
		}

		err = api.ErrNotAvailable
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()

	if err == nil {
		return res.Odometer.Mileage, nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err == nil {
		for _, e := range res.Energy {
			if e.Type != "Electric" {
				continue
			}

			return e.UpdatedAt.Add(e.Charging.RemainingTime.Duration), nil
		}

		err = api.ErrNotAvailable
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err == nil {
		for _, e := range res.Energy {
			if e.Type != "Electric" {
				continue
			}

			status := api.StatusA

			if e.Charging.Plugged {
				status = api.StatusB

				if strings.ToLower(e.Charging.Status) == "inprogress" {
					status = api.StatusC
				}
			}

			return status, nil
		}

		err = api.ErrNotAvailable
	}

	return api.StatusNone, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	if err == nil {
		active := strings.ToLower(res.Preconditionning.AirConditioning.Status) == "enabled"
		return active, nil
	}

	return false, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	if err == nil {
		if coord := res.LastPosition.Geometry.Coordinates; len(coord) >= 2 {
			return coord[0], coord[1], nil
		}
		return 0, 0, api.ErrNotAvailable
	}

	return 0, 0, err
}
