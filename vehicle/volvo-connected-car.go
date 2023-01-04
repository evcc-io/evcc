package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/volvo/connectedcar"
)

// VolvoConnectedCar is an api.Vehicle implementation for Volvo Connected Car vehicles
type VolvoConnectedCar struct {
	*embed
	// api.ProviderLogin
	*connectedcar.Provider
}

func init() {
	registry.Add("volvo-connected-car", NewVolvoConnectedCarFromConfig)
}

// NewVolvoConnectedCarFromConfig creates a new VolvoConnectedCar vehicle
func NewVolvoConnectedCarFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		// ClientID, ClientSecret string
		// Sandbox                bool
		VccApiKey string
		Cache     time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// if cc.ClientID == "" && cc.ClientSecret == "" {
	// 	return nil, errors.New("missing credentials")
	// }

	// var options []VolvoConnectedCar.IdentityOptions

	// TODO Load tokens from a persistence storage and use those during startup
	// e.g. persistence.Load("key")
	// if tokens != nil {
	// 	options = append(options, VolvoConnectedCar.WithToken(&oauth2.Token{
	// 		AccessToken:  tokens.Access,
	// 		RefreshToken: tokens.Refresh,
	// 		Expiry:       tokens.Expiry,
	// 	}))
	// }

	log := util.NewLogger("volvo-cc").Redact(cc.User, cc.Password, cc.VIN, cc.VccApiKey)

	// identity, err := connectedcar.NewIdentity(log, cc.ClientID, cc.ClientSecret)
	identity, err := connectedcar.NewIdentity(log)
	if err != nil {
		return nil, err
	}

	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, err
	}

	_ = identity
	// api := connectedcar.NewAPI(log, identity, cc.Sandbox)
	api := connectedcar.NewAPI(log, identity, cc.VccApiKey)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	v := &VolvoConnectedCar{
		embed:    &cc.embed,
		Provider: connectedcar.NewProvider(api, cc.VIN, cc.Cache),
		// ProviderLogin: identity, // expose the OAuth2 login
	}

	return v, nil
}
