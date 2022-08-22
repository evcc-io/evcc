package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/mercedes"
)

// Mercedes is an api.Vehicle implementation for Mercedes cars
type Mercedes struct {
	*embed
	api.AuthProvider
	*mercedes.Provider
}

func init() {
	registry.Add("mercedes", NewMercedesFromConfig)
}

// NewMercedesFromConfig creates a new Mercedes vehicle
func NewMercedesFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
		ClientID, ClientSecret string
		VIN                    string
		Sandbox                bool
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" && cc.ClientSecret == "" {
		return nil, errors.New("missing credentials")
	}

	var options []mercedes.IdentityOptions

	// TODO Load tokens from a persistence storage and use those during startup
	// e.g. persistence.Load("key")
	// if tokens != nil {
	// 	options = append(options, mercedes.WithToken(&oauth2.Token{
	// 		AccessToken:  tokens.Access,
	// 		RefreshToken: tokens.Refresh,
	// 		Expiry:       tokens.Expiry,
	// 	}))
	// }

	log := util.NewLogger("mercedes")

	// TODO session secret from config/persistence
	identity, err := mercedes.NewIdentity(log, cc.ClientID, cc.ClientSecret, options...)
	if err != nil {
		return nil, err
	}

	api := mercedes.NewAPI(log, identity, cc.Sandbox)

	v := &Mercedes{
		embed:        &cc.embed,
		Provider:     mercedes.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache),
		AuthProvider: identity, // expose the OAuth2 login
	}

	return v, nil
}
