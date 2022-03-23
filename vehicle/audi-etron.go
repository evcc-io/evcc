package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/audi/etron"
	"github.com/evcc-io/evcc/vehicle/id"
	"github.com/evcc-io/evcc/vehicle/vag/aazsproxy"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.vw-connect
// https://github.com/arjenvrh/audi_connect_ha/blob/master/custom_components/audiconnect/audi_services.py

// Etron is an api.Vehicle implementation for Audi eTron cars
type Etron struct {
	*embed
	*id.Provider // provides the api implementations
}

func init() {
	registry.Add("etron", NewEtronFromConfig)
}

// NewEtronFromConfig creates a new vehicle
func NewEtronFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Etron{
		embed: &cc.embed,
	}

	log := util.NewLogger("etron").Redact(cc.User, cc.Password, cc.VIN)

	// // add code challenge
	// cvc, _ := cv.CreateCodeVerifier()

	// q := url.Values{
	// 	"code_challenge_method": {"S256"},
	// 	"code_challenge":        {cvc.CodeChallengeS256()},
	// }

	// for k, v := range etron.AuthParams {
	// 	q[k] = v
	// }

	vwi := vwidentity.New(log)
	uri := vwidentity.LoginURL(vwidentity.Endpoint.AuthURL, etron.AuthParams)
	q, err := vwi.Login(uri, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	azs := aazsproxy.New(log)
	token, err := azs.Exchange(q)
	if err != nil {
		return nil, err
	}

	api := etron.NewAPI(log, azs.TokenSource(token))

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		// TODO build token source
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: q.Get("id_token")})
		api := id.NewAPI(log, ts)

		api.Client.Timeout = cc.Timeout

		v.Provider = id.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
