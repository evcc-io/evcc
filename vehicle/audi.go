package vehicle

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/oidc"
	"github.com/andig/evcc/vehicle/vw"
	"golang.org/x/net/publicsuffix"
)

// https://github.com/davidgiga1993/AudiAPI

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*request.Helper
	user, password     string
	tokens             oidc.Tokens
	api                *vw.API
	*vw.Implementation // provides the api implementations
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

const audiClientID = "77869e21-e30a-4a92-b016-48ab7d3db1d8"

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("audi")

	v := &Audi{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
	}

	v.api = vw.NewAPI(v.Helper, &v.tokens, v.authFlow, v.refreshHeaders, strings.ToUpper(cc.VIN), "Audi", "DE")
	v.Implementation = vw.NewImplementation(v.api, cc.Cache)

	var err error
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and don't follow redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	if err == nil {
		err = v.authFlow()
	}

	if err == nil && cc.VIN == "" {
		v.api.VIN, err = findVehicle(v.api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.api.VIN)
		}
	}

	return v, err
}

func (v *Audi) authFlow() error {
	var req *http.Request

	const clientID = "09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"
	const redirectURI = "myaudi:///"

	query := url.Values(map[string][]string{
		"response_type": {"id_token token"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"scope":         {"openid profile mbb vin badge birthdate nickname email address phone name picture"},
		"state":         {vw.RandomString(43)},
		"nonce":         {vw.RandomString(43)},
		"prompt":        {"login"},
		"ui_locales":    {"de-DE"},
	})

	identity := &vw.Identity{Client: v.Client}
	idToken, err := identity.Login(query, v.user, v.password)

	if err == nil {
		data := url.Values(map[string][]string{
			"grant_type": {"id_token"},
			"scope":      {"sc2:fal"},
			"token":      {idToken},
		})

		req, err = request.New(http.MethodPost, vw.OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"X-Client-Id":  audiClientID,
		})
		if err == nil {
			if err = v.DoJSON(req, &v.tokens); err == nil && v.tokens.AccessToken == "" {
				err = errors.New("missing access token")
			}
		}
	}

	return err
}

func (v *Audi) refreshHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"X-App-Version": "3.14.0",
		"X-App-Name":    "myAudi",
		"X-Client-Id":   audiClientID,
	}
}
