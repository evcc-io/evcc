package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/mercedes"
	"golang.org/x/oauth2"
)

// Mercedes is an api.Vehicle implementation for Mercedes cars
type Mercedes struct {
	*embed
	*mercedes.Identity
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
		Tokens                 Tokens
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// TODO: cc.Tokens should be moved to a persistence storage
	// e.g. persistence.Load("key")

	if cc.ClientID == "" && cc.Tokens.Refresh == "" {
		return nil, errors.New("missing credentials")
	}

	var options []mercedes.ClientOption
	if cc.Tokens.Refresh != "" {
		options = append(options, mercedes.WithToken(&oauth2.Token{
			AccessToken:  cc.Tokens.Access,
			RefreshToken: cc.Tokens.Refresh,
			Expiry:       time.Now(),
		}))
	}

	log := util.NewLogger("mercedes")

	updateC := make(chan struct{})

	// TODO: session secret from config/persistence
	identity, err := mercedes.NewIdentity(log, cc.ClientID, cc.ClientSecret, updateC, options...)
	if err != nil {
		return nil, err
	}

	api := mercedes.NewAPI(log, identity, updateC)

	v := &Mercedes{
		embed:    &cc.embed,
		Identity: identity,
		Provider: mercedes.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache),
	}

	return v, nil
}
