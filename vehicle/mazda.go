package vehicle

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	mazda "github.com/tah78/mazda-go"
)

// https://github.com/fano0001/home-assistant-mazda-6e

// Mazda is an api.Vehicle implementation for the Mazda 6e and CX-6e. Both models
// share the app backend. Login happens in the ui via evcc's provider auth.
type Mazda struct {
	*embed
	client  *mazda.Client
	mu      sync.Mutex
	vin     string
	vehicle *mazda.Vehicle
	statusG func() (mazda.Status, error)
}

func init() {
	registry.AddCtx("mazda", NewMazdaFromConfig)
}

// NewMazdaFromConfig creates a new vehicle
func NewMazdaFromConfig(_ context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("mazda").Redact(cc.VIN)

	identity, err := newMazdaIdentity(log, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	v := &Mazda{
		embed:  &cc.embed,
		client: identity.client,
		vin:    cc.VIN,
	}

	statusC := util.ResettableCached(v.status, cc.Cache)
	v.statusG = func() (mazda.Status, error) {
		res, err := statusC.Get()
		if err != nil {
			// do not cache a login error across a successful login
			if _, ok := errors.AsType[*api.ErrLoginRequired](err); ok {
				statusC.Reset()
			}
		}
		return res, err
	}

	return v, nil
}

func (v *Mazda) status() (mazda.Status, error) {
	veh, err := v.resolve()
	if err != nil {
		return mazda.Status{}, err
	}

	return v.client.Status(context.Background(), veh.EffectiveID())
}

// resolve returns the configured vehicle, resolved and cached on first use.
func (v *Mazda) resolve() (*mazda.Vehicle, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.vehicle != nil {
		return v.vehicle, nil
	}

	veh, err := ensureVehicleEx(v.vin, func() ([]mazda.Vehicle, error) {
		return v.client.Vehicles(context.Background())
	}, func(veh mazda.Vehicle) (string, error) {
		return veh.VIN, nil
	})
	if err != nil {
		return nil, err
	}

	v.vehicle = &veh

	return v.vehicle, nil
}

var _ api.Battery = (*Mazda)(nil)

// Soc implements the api.Battery interface.
func (v *Mazda) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if res.VehicleStatus.Soc == nil {
		return 0, api.ErrNotAvailable
	}

	return *res.VehicleStatus.Soc, nil
}

var _ api.ChargeState = (*Mazda)(nil)

// Status implements the api.ChargeState interface. The vehicle reports the plug
// separately from the charging state.
func (v *Mazda) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	if res.Charge.ConStatus == nil {
		return api.StatusNone, api.ErrNotAvailable
	}

	if *res.Charge.ConStatus == mazda.ChargeDisconnected {
		return api.StatusA, nil
	}

	if res.Charge.Status != nil && *res.Charge.Status == mazda.ChargeStatusCharging {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

var _ api.VehicleRange = (*Mazda)(nil)

// Range implements the api.VehicleRange interface.
func (v *Mazda) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if res.VehicleStatus.Range == nil {
		return 0, api.ErrNotAvailable
	}

	return int64(*res.VehicleStatus.Range), nil
}

var _ api.VehicleOdometer = (*Mazda)(nil)

// Odometer implements the api.VehicleOdometer interface.
func (v *Mazda) Odometer() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	if res.VehicleStatus.Odometer == nil {
		return 0, api.ErrNotAvailable
	}

	return *res.VehicleStatus.Odometer, nil
}

var _ api.VehicleClimater = (*Mazda)(nil)

// Climater implements the api.VehicleClimater interface.
func (v *Mazda) Climater() (bool, error) {
	res, err := v.statusG()
	if err != nil {
		return false, err
	}

	if res.HVAC.AcStatus == nil {
		return false, api.ErrNotAvailable
	}

	return *res.HVAC.AcStatus != 0, nil
}
