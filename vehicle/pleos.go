package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/pleos"
)

// Pleos is an api.Vehicle implementation for Hyundai, Kia and Genesis cars
// using the Pleos Vehicle Data API (EU Data Act)
type Pleos struct {
	*embed
	*pleos.Provider
}

func init() {
	registry.Add("pleos", NewPleosFromConfig)
}

// NewPleosFromConfig creates a new vehicle
func NewPleosFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
		Brand                  string
		ClientID, ClientSecret string
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: 15 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" || cc.ClientSecret == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("pleos").Redact(cc.ClientID, cc.ClientSecret, cc.VIN)

	// an api key is issued per manufacturer, so the brand can be probed
	brands := []string{"hyundai", "kia", "genesis"}
	if cc.Brand != "" {
		brands = []string{strings.ToLower(cc.Brand)}
	}

	var (
		api *pleos.API
		err error
	)
	for _, brand := range brands {
		identity := pleos.NewIdentity(log, brand, cc.ClientID, cc.ClientSecret)
		if _, err = identity.Token(); err == nil {
			api = pleos.NewAPI(log, identity, brand)
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	v := &Pleos{
		embed: &cc.embed,
	}

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v pleos.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)

	if err == nil {
		v.fromVehicle(vehicle.Title(), 0)
		v.Provider = pleos.NewProvider(api, vehicle.VIN, cc.Cache)
	}

	return v, err
}
