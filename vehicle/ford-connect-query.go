package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/ford/connect"
	"github.com/evcc-io/evcc/vehicle/ford/query"
	"golang.org/x/oauth2"
)

// https://developer.ford.com/apis/fordconnect-query

// FordConnectQuery is an api.Vehicle implementation for Ford cars
type FordConnectQuery struct {
	*embed
	*connect.Provider
}

func init() {
	registry.Add("ford-connect-query", NewFordConnectQueryFromConfig)
}

// NewFordConnectQueryFromConfig creates a new vehicle
func NewFordConnectQueryFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed       `mapstructure:",squash"`
		Credentials ClientCredentials
		RedirectURI string
		VIN         string
		Cache       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &FordConnectQuery{
		embed: &cc.embed,
	}

	if err := cc.Credentials.Error(); err != nil {
		return nil, err
	}

	oc := query.Config
	oc.ClientID = cc.Credentials.ID
	oc.ClientSecret = cc.Credentials.Secret
	oc.RedirectURL = cc.RedirectURI

	log := util.NewLogger("ford").Redact(cc.VIN, cc.Credentials.ID, cc.Credentials.Secret)
	ts, err := auth.NewOauth(context.Background(), "Ford", cc.embed.GetTitle(), &oc,
		auth.WithOauthAuthCodeOptionsOption(oauth2.SetAuthURLParam("p", "B2C_1A_FCON_AUTHORIZE")),
		// auth.WithOauthDeviceFlowOption(),
		// auth.WithTokenRetrieverOption(func(data string, res *oauth2.Token) error {
		// 	var token cardata.Token
		// 	if err := json.Unmarshal([]byte(data), &token); err != nil {
		// 		return err
		// 	}
		// 	*res = *token.TokenEx()
		// 	return nil
		// }),
		// auth.WithTokenStorerOption(func(token *oauth2.Token) any {
		// 	return cardata.Token{
		// 		Token:   token,
		// 		IdToken: cardata.TokenExtra(token, "id_token"),
		// 		Gcid:    cardata.TokenExtra(token, "gcid"),
		// 	}
		// })
	)

	if err != nil {
		return nil, err
	}

	api := connect.NewAPI(log, ts)

	vehicle, err := ensureVehicleEx(cc.VIN, api.Vehicles, func(v connect.Vehicle) (string, error) {
		return api.VIN(v.VehicleID)
	})

	if err == nil {
		v.fromVehicle(vehicle.NickName, 0)
		v.Provider = connect.NewProvider(api, vehicle.VehicleID, cc.Cache)
	}

	return v, err
}
