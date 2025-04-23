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
		embed       `mapstructure:",squash"`
		VIN         string
		VccApiKey   string
		Credentials ClientCredentials
		Tokens      Tokens
		Cache       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("volvo-connected").Redact(cc.VIN, cc.VccApiKey, cc.Tokens.Access, cc.Tokens.Refresh)

	oc := connected.Oauth2Config(cc.Credentials.ID, cc.Credentials.Secret)

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	ts, err := connected.NewIdentity(log, oc, token)
	if err != nil {
		return nil, err
	}

	api := connected.NewAPI(log, ts, cc.VccApiKey)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	v := &VolvoConnected{
		embed:    &cc.embed,
		Provider: connected.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, err
}
