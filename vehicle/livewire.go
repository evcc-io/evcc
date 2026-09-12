package vehicle

import (
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/livewire"
)

// LiveWire is an api.Vehicle implementation for LiveWire motorcycles
type LiveWire struct {
	*embed
	*livewire.Provider
}

func init() {
	registry.Add("livewire", NewLiveWireFromConfig)
}

// NewLiveWireFromConfig creates a new vehicle
func NewLiveWireFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		DeviceUUID          string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	// the device uuid is created and paired with the motorcycle by the pairing script
	if cc.DeviceUUID == "" {
		return nil, errors.New("missing deviceUUID, run the pairing script first")
	}
	if _, err := uuid.Parse(cc.DeviceUUID); err != nil {
		return nil, fmt.Errorf("invalid deviceUUID: %w", err)
	}

	log := util.NewLogger("livewire").Redact(cc.User, cc.Password, cc.DeviceUUID)

	identity := livewire.NewIdentity(log, cc.User, cc.Password, cc.DeviceUUID)
	if err := identity.Login(); err != nil {
		return nil, err
	}

	res := livewire.NewAPI(log, identity)

	vehicle, err := ensureVehicleEx(cc.VIN, res.Vehicles,
		func(v livewire.Bike) (string, error) {
			return v.VIN, nil
		},
	)
	if err != nil {
		return nil, err
	}

	if !vehicle.PairingStatus {
		return nil, errors.New("deviceUUID is not paired with the motorcycle, run the pairing script")
	}

	v := &LiveWire{
		embed:    &cc.embed,
		Provider: livewire.NewProvider(res, vehicle.ID, cc.Cache),
	}

	return v, nil
}
