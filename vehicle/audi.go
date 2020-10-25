package vehicle

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/vw"
	"golang.org/x/net/publicsuffix"
)

// https://github.com/davidgiga1993/AudiAPI

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*request.Helper
	user, password     string
	tokens             vw.Tokens
	api                *vw.API
	*vw.Implementation // provides the api implementations
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

// AudiHashSecret is used for obtaining the X-QMauth header hash value from the current timestamp
var AudiHashSecret = "not contained in repo due to legal concerns"

const audiOAuthClientID = "77869e21-e30a-4a92-b016-48ab7d3db1d8"

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
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
	var err error
	var uri string
	var req *http.Request
	var resp *http.Response
	var tokens vw.Tokens

	const clientID = "09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"
	const clientIDAlias = "934928ef" // "09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"
	const redirectURI = "myaudi:///"

	query := url.Values(map[string][]string{
		"response_type": {"code"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"scope":         {"address profile badge birthdate birthplace nationalIdentifier nationality profession email vin phone nickname name picture mbb gallery openid"},
		"state":         {"7f8260b5-682f-4db8-b171-50a5189a1c08"},
		"nonce":         {"7f8260b5-682f-4db8-b171-50a5189a1c08"},
		"prompt":        {"login"},
		"ui_locales":    {"de-DE"},
	})

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" + query.Encode()
	if err == nil {
		identity := &vw.Identity{Client: v.Client}
		resp, err = identity.Login(uri, v.user, v.password)
	}

	if err == nil {
		var code string
		if location, err := url.Parse(resp.Header.Get("Location")); err == nil {
			code = location.Query().Get("code")
		}

		data := url.Values(map[string][]string{
			"client_id":     {clientID},
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {redirectURI},
			"response_type": {"token id_token"},
		})

		var hash string
		var secret []byte
		if secret, err = hex.DecodeString(AudiHashSecret); err == nil {
			// timestamp rounded to 100s precision
			ts := strconv.FormatInt(time.Now().Unix()/100, 10)

			mac := hmac.New(sha256.New, secret)
			_, err = mac.Write([]byte(ts))

			hash = fmt.Sprintf("v1:%s:%0x", clientIDAlias, mac.Sum(nil))
		}

		if err == nil {
			uri = "https://app-api.my.audi.com/myaudiappidk/v1/emea/token"
			req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"X-QMAuth":     hash,
			})
		}

		if err == nil {
			if err = v.DoJSON(req, &tokens); err == nil && tokens.IDToken == "" {
				err = errors.New("missing id token (1)")
			}
		}
	}

	if err == nil {
		data := url.Values(map[string][]string{
			"grant_type": {"id_token"},
			"scope":      {"sc2:fal"},
			"token":      {tokens.IDToken},
		})

		req, err = request.New(http.MethodPost, vw.OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"X-Client-Id":  audiOAuthClientID,
		})
		if err == nil {
			if err = v.DoJSON(req, &v.tokens); err == nil && tokens.IDToken == "" {
				err = errors.New("missing id token (2)")
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
		"X-Client-Id":   audiOAuthClientID,
	}
}
