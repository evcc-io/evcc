package vehicle

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bmw/cardata"
	"golang.org/x/oauth2"
)

// Cardata is an api.Vehicle implementation for BMW and Mini cars
type Cardata struct {
	*embed
	*cardata.Provider // provides the api implementations
}

func init() {
	registry.AddCtx("cardata", NewCardataFromConfig)
}

// NewCardataFromConfig creates a new BMW/Mini CarData vehicle
func NewCardataFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	var cc struct {
		embed         `mapstructure:",squash"`
		ClientID, VIN string
		Cache         time.Duration // 50 requests per day
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.VIN == "" {
		return nil, errors.New("missing vin")
	}

	if cc.ClientID == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Cache == 0 {
		// for non-streaming use 15m, access controlled by loadpoint
		isStreaming := slices.Contains(cc.embed.Features(), api.Streaming)
		cc.Cache = map[bool]time.Duration{false: 15 * time.Minute, true: 30 * time.Minute}[isStreaming]
	}

	v := &Cardata{
		embed: &cc.embed,
	}

	oc := cardata.Config
	oc.ClientID = cc.ClientID

	log := util.NewLogger("cardata").Redact(cc.ClientID, cc.VIN)

	ts, err := auth.NewOauth(context.Background(), "BMW/Mini", cc.embed.GetTitle(), &oc,
		auth.WithOauthDeviceFlowOption(),
		auth.WithTokenRetrieverOption(func(data string, res *oauth2.Token) error {
			var token cardata.Token
			if err := json.Unmarshal([]byte(data), &token); err != nil {
				return err
			}
			*res = *token.TokenEx()
			return nil
		}),
		auth.WithTokenStorerOption(func(token *oauth2.Token) any {
			return cardata.Token{
				Token:   token,
				IdToken: cardata.TokenExtra(token, "id_token"),
				Gcid:    cardata.TokenExtra(token, "gcid"),
			}
		}))
	if err != nil {
		return nil, err
	}

	api := cardata.NewAPI(log, ts)

	v.Provider = cardata.NewProvider(ctx, log, api, ts, cc.ClientID, cc.VIN, cc.Cache)

	return v, nil
}
