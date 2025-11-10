package vehicle

import (
	"context"
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/volvo/connected"
)

// VolvoConnected is an api.Vehicle implementation for Volvo Connected Car vehicles
type VolvoConnected struct {
	*embed
	*connected.Provider
}

func init() {
	registry.AddCtx("volvo-connected", NewVolvoConnectedFromConfig)
}

// NewVolvoConnectedFromConfig creates a new VolvoConnected vehicle
func NewVolvoConnectedFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
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

	if cc.VccApiKey == "" {
		return nil, errors.New("missing vccapikey")
	}

	if cc.VIN == "" {
		return nil, errors.New("missing vin")
	}

	if err := cc.Credentials.Error(); err != nil {
		return nil, err
	}

	log := util.NewLogger("volvo-connected").Redact(cc.VIN, cc.Credentials.ID, cc.Credentials.Secret, cc.VccApiKey)

	oc := connected.Oauth2Config(cc.Credentials.ID, cc.Credentials.Secret, cc.RedirectUri)
	ts, err := auth.NewOauth(ctx, "Volvo", cc.embed.GetTitle(), oc)
	if err != nil {
		return nil, err
	}

	api := connected.NewAPI(log, cc.VccApiKey, ts)

	v := &VolvoConnected{
		embed:    &cc.embed,
		Provider: connected.NewProvider(api, ts, cc.VIN, cc.Cache),
	}

	return v, err
}
