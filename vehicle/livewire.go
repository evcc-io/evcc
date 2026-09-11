package vehicle

import (
	"fmt"
	"time"
	"uuid"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
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

	deviceUUID, err := livewireDeviceUUID(cc.User, cc.DeviceUUID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("livewire").Redact(cc.User, cc.Password, deviceUUID)

	identity := livewire.NewIdentity(log, cc.User, cc.Password, deviceUUID)
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
		return nil, fmt.Errorf("device %s is not paired with the motorcycle, pair it once at the bike", deviceUUID)
	}

	v := &LiveWire{
		embed:    &cc.embed,
		Provider: livewire.NewProvider(res, vehicle.ID, cc.Cache),
	}

	return v, nil
}

// livewireDeviceUUID returns the device uuid paired with the motorcycle. The
// configured value seeds it, otherwise the persisted one is reused and only
// generated once: a fresh uuid would need pairing at the bike again.
func livewireDeviceUUID(user, configured string) (string, error) {
	key := fmt.Sprintf("vehicle.livewire.%s.deviceUUID", user)

	if configured != "" {
		if _, err := uuid.Parse(configured); err != nil {
			return "", fmt.Errorf("invalid deviceUUID: %w", err)
		}
		settings.SetString(key, configured)
		return configured, nil
	}

	if stored, err := settings.String(key); err == nil && stored != "" {
		return stored, nil
	}

	generated := uuid.New().String()
	settings.SetString(key, generated)

	return generated, nil
}
