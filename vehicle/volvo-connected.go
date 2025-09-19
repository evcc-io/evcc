package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/volvo/connected"
	"golang.org/x/oauth2"
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
		RedirectUri string
		Cache       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("volvo-connected").Redact(cc.VIN, cc.VccApiKey)

	// create oauth2 config
	config := connected.Oauth2Config(cc.Credentials.ID, cc.Credentials.Secret, cc.RedirectUri)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))
	authorizer, err := auth.NewOauth(ctx, *config, cc.embed.GetTitle())
	if err != nil {
		return nil, err
	}

	api := connected.NewAPI(log, cc.VccApiKey, authorizer)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	v := &VolvoConnected{
		embed:    &cc.embed,
		Provider: connected.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, err
}
