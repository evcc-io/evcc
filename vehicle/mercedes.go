package vehicle

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/mercedes"
)

// Mercedes is an api.Vehicle implementation for Mercedes-Benz cars
type Mercedes struct {
	*embed
	*mercedes.Provider
}

func init() {
	registry.Add("mercedes", NewMercedesFromConfig)
}

// NewMercedesFromConfig creates a new vehicle
func NewMercedesFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed   `mapstructure:",squash"`
		Tokens  Tokens
		Account string
		VIN     string
		Cache   time.Duration
		Region  string
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	if token.AccessToken == "" || token.RefreshToken == "" {
		return nil, errors.New("Access/Refresh Token not defined. Please generate the token with 'evcc token mercedes' and add it to the configuration file.")
	}

	log := util.NewLogger("mercedes").Redact(cc.Tokens.Access, cc.Tokens.Refresh)
	identity, err := mercedes.NewIdentity(log, token, cc.Account, cc.Region)

	v := &Mercedes{
		embed: &cc.embed,
	}

	api := mercedes.NewAPI(log, identity)

	if err == nil {
		v.Provider = mercedes.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
