package vehicle

import (
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/audi/etron"
	"github.com/evcc-io/evcc/vehicle/id"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/aazsproxy"
	"github.com/evcc-io/evcc/vehicle/vag/idkproxy"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
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

	// get initial VW identity id_token
	q := urlvalues.Copy(etron.AuthParams)
	verify := vag.ChallengeAndVerifier(q)

	vwi := vwidentity.New(log)
	uri := vwidentity.LoginURL(vwidentity.Endpoint.AuthURL, q)
	q, err := vwi.Login(uri, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	verify(q)

	// exchange initial VW identity id_token for Audi IDK token
	idk := idkproxy.New(log, etron.IDKParams)
	token, err := idk.Exchange(q)
	if err != nil {
		return nil, err
	}

	// refreshing IDK token source
	its := idk.TokenSource(token)
	azs := aazsproxy.New(log)

	// create AAZS token source that refreshes using IDK token
	ats := vag.MetaTokenSource(func() (*vag.Token, error) {
		// get IDK token from refreshing IDK token source
		itoken, err := its.TokenEx()
		if err != nil {
			return nil, err
		}

		// exchange IDK id_token for AAZS token
		atoken, err := azs.Exchange(url.Values{"id_token": {itoken.IDToken}})
		if err != nil {
			return nil, err
		}

		return atoken, err

		// produce tokens from AAZS token source
	}, azs.TokenSource)

	// use the etron API for list of vehicles
	api := etron.NewAPI(log, ats)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		api := id.NewAPI(log, vag.IDTokenSource(its))
		api.Client.Timeout = cc.Timeout

		v.Provider = id.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
