package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/volvo/connected"
)

// VolvoConnected is an api.Vehicle implementation for Volvo Connected Car vehicles
type VolvoConnected struct {
	*embed
	*connected.Provider
}

func init() {
	registry.Add("volvo-connected", NewVolvoConnectedFromConfig)
}

// NewVolvoConnectedFromConfig creates a new VolvoConnected vehicle
func NewVolvoConnectedFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	// if cc.ClientID == "" && cc.ClientSecret == "" {
	// 	return nil, errors.New("missing credentials")
	// }

	// var options []VolvoConnected.IdentityOptions

	// TODO Load tokens from a persistence storage and use those during startup
	// e.g. persistence.Load("key")
	// if tokens != nil {
	// 	options = append(options, VolvoConnected.WithToken(&oauth2.Token{
	// 		AccessToken:  tokens.Access,
	// 		RefreshToken: tokens.Refresh,
	// 		Expiry:       tokens.Expiry,
	// 	}))
	// }

	log := util.NewLogger("volvo-cc").Redact(cc.User, cc.Password, cc.VIN, cc.VccApiKey)

	// identity, err := connected.NewIdentity(log, cc.ClientID, cc.ClientSecret)
	identity, err := connected.NewIdentity(log)
	if err != nil {
		return nil, err
	}

	ts, err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	// api := connected.NewAPI(log, identity, cc.Sandbox)
	api := connected.NewAPI(log, ts, cc.VccApiKey)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	v := &VolvoConnected{
		embed:    &cc.embed,
		Provider: connected.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, err
}
