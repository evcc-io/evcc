package psa

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Provider is an api.Vehicle implementation for PSA cars
type Provider struct {
	statusG func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vid string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vid)
		}, cache).InterfaceGetter(),
	}
	return impl
}

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		for _, e := range res.Energy {
			return float64(e.Level), nil
		}

		err = api.ErrNotAvailable
	}

	return 0, err
}

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		for _, e := range res.Energy {
			return int64(e.Autonomy), nil
		}

		err = api.ErrNotAvailable
	}

	return 0, err
}

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		if len(res.Energy) == 1 {
			e := res.Energy[0]
			return e.UpdatedAt.Add(e.Charging.RemainingTime.Duration), nil
		}

		err = api.ErrNotAvailable
	}

	return time.Time{}, err
}

// Status implements the api.VehicleStatus interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		if len(res.Energy) == 1 {
			status := api.StatusA

			e := res.Energy[0]
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

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		active := strings.ToLower(res.Preconditionning.AirConditioning.Status) != "disabled"
		return active, 20, 20, nil
	}

	return active, outsideTemp, targetTemp, err
}
