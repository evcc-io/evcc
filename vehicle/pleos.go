package vehicle

import (
	"fmt"
	"slices"
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

	brand := strings.ToLower(cc.Brand)
	if !slices.Contains([]string{"hyundai", "kia", "genesis"}, brand) {
		return nil, fmt.Errorf("invalid brand: %s", cc.Brand)
	}

	if cc.VIN == "" {
		return nil, fmt.Errorf("missing vin")
	}

	log := util.NewLogger("pleos").Redact(cc.ClientID, cc.ClientSecret, cc.VIN)

	identity := pleos.NewIdentity(log, brand, cc.ClientID, cc.ClientSecret)
	api := pleos.NewAPI(log, identity, brand)

	v := &Pleos{
		embed:    &cc.embed,
		Provider: pleos.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache),
	}

	return v, nil
}
