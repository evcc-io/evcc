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

// https://github.com/trocotronic/weconnect

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*request.Helper
	user, password     string
	clientID           string
	tokens             oidc.Tokens
	api                *vw.API
	*vw.Implementation // provides the api implementations
}

func init() {
	registry.Add("vw", NewVWFromConfig)
}

const vwClientID = "38761134-34d0-41f3-9a73-c4be88d7d337"

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("vw")

	v := &VW{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
	}

	v.api = vw.NewAPI(v.Helper, &v.tokens, v.authFlow, v.refreshHeaders, strings.ToUpper(cc.VIN), "VW", "DE")
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

func (v *VW) authFlow() error {
	var req *http.Request

	const clientID = "9496332b-ea03-4091-a224-8c746b885068@apps_vw-dilab_com"
	const redirectURI = "carnet://identity-kit/login"

	query := url.Values(map[string][]string{
		"response_type": {"id_token token"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"scope":         {"openid profile mbb cars birthdate nickname address phone"},
		"state":         {vw.RandomString(43)},
		"nonce":         {vw.RandomString(43)},
		"prompt":        {"login"},
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
			"Content-type": "application/x-www-form-urlencoded",
			"X-Client-Id":  vwClientID,
		})

		if err == nil {
			if err = v.DoJSON(req, &v.tokens); err == nil && v.tokens.AccessToken == "" {
				err = errors.New("missing access token")
			}
		}
	}

	return err
}

func (v *VW) refreshHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  vwClientID,
	}
}
