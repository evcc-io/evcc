package vehicle

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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
func NewCardataFromConfig(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed         `mapstructure:",squash"`
		ClientID, VIN string
		Cache         time.Duration
	}{
		Cache: 30 * time.Minute, // 50 requests per day
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Cardata{
		embed: &cc.embed,
	}

	oc := cardata.Config
	oc.ClientID = cc.ClientID

	log := util.NewLogger("cardata").Redact(cc.ClientID)

	ts, err := auth.NewOauth(context.Background(), "BMW/Mini CarData", &oc,
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

	is429 := func(err error) bool {
		se := new(request.StatusError)
		return errors.As(err, &se) && se.StatusCode() == http.StatusTooManyRequests
	}

	vehicle, err := ensureVehicle(
		cc.VIN, api.Vehicles,
	)
	if err != nil && (cc.VIN == "" || !is429(err)) {
		return nil, err
	}

	v.Provider = cardata.NewProvider(ctx, log, api, ts, vehicle)

	return v, nil
}
