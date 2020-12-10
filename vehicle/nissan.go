package vehicle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/kamereon"
	"github.com/andig/evcc/vehicle/oidc"
)

// Credits to
//   https://github.com/Tobiaswk/dartnissanconnect
//   https://github.com/mitchellrj/kamereon-python
//   https://gitlab.com/tobiaswkjeldsen/carwingsflutter

// OAuth base url
// 	 https://prod.eu.auth.kamereon.org/kauth/oauth2/a-ncb-prod/.well-known/openid-configuration

// api constants
const (
	nissanAPIVersion         = "protocol=1.0,resource=2.1"
	nissanClientID           = "a-ncb-prod-android"
	nissanClientSecret       = "3LBs0yOx2XO-3m4mMRW27rKeJzskhfWF0A8KUtnim8i/qYQPl8ZItp3IaqJXaYj_"
	nissanScope              = "openid profile vehicles"
	nissanAuthBaseURL        = "https://prod.eu.auth.kamereon.org/kauth"
	nissanRealm              = "a-ncb-prod"
	nissanRedirectURI        = "org.kamereon.service.nci:/oauth2redirect"
	nissanCarAdapterBaseURL  = "https://alliance-platform-caradapter-prod.apps.eu.kamereon.io/car-adapter"
	nissanUserAdapterBaseURL = "https://alliance-platform-usersadapter-prod.apps.eu.kamereon.io/user-adapter"
	nissanUserBaseURL        = "https://nci-bff-web-prod.apps.eu.kamereon.io/bff-web"
)

// Nissan is an api.Vehicle implementation for Nissan cars
type Nissan struct {
	*embed
	*request.Helper
	log                 *util.Logger
	user, password, vin string
	userID              string
	tokens              oidc.Tokens
	*kamereon.API
}

func init() {
	registry.Add("nissan", NewNissanFromConfig)
}

// NewNissanFromConfig creates a new vehicle
func NewNissanFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                       string
		Capacity                    int64
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{
		Region: "de_DE",
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("nissan")

	v := &Nissan{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		log:      log,
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	err := v.authFlow()

	if err == nil && cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles(v.userID))
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	v.API = kamereon.New(provider.NewCached(v.batteryAPI, cc.Cache).InterfaceGetter())

	return v, err
}

type nissanAuth struct {
	AuthID    string               `json:"authId"`
	Template  string               `json:"template"`
	Stage     string               `json:"stage"`
	Header    string               `json:"header"`
	Callbacks []nissanAuthCallback `json:"callbacks"`
}

type nissanAuthCallback struct {
	Type   string                    `json:"type"`
	Output []nissanAuthCallbackValue `json:"output"`
	Input  []nissanAuthCallbackValue `json:"input"`
}

type nissanAuthCallbackValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type nissanToken struct {
	TokenID    string `json:"tokenId"`
	SuccessURL string `json:"successUrl"`
	Realm      string `json:"realm"`
}

func (v *Nissan) authFlow() error {
	uri := fmt.Sprintf("%s/json/realms/root/realms/%s/authenticate", nissanAuthBaseURL, nissanRealm)

	req, err := request.New(http.MethodPost, uri, nil, map[string]string{
		"Accept-Api-Version": nissanAPIVersion,
		"X-Username":         "anonymous",
		"X-Password":         "anonymous",
		"Accept":             "application/json",
	})

	var oauth nissanToken
	var resp *http.Response
	var code string

	if err == nil {
		var res nissanAuth
		if err = v.DoJSON(req, &res); err != nil {
			return err
		}

		for id, cb := range res.Callbacks {
			switch cb.Type {
			case "NameCallback":
				res.Callbacks[id].Input[0].Value = v.user
			case "PasswordCallback":
				res.Callbacks[id].Input[0].Value = v.password
			}
		}

		var body []byte
		body, err = json.Marshal(res)

		if err == nil {
			req, err = request.New(http.MethodPost, uri, bytes.NewReader(body), map[string]string{
				"Content-type":       "application/json",
				"Accept-Api-Version": nissanAPIVersion,
				"X-Username":         "anonymous",
				"X-Password":         "anonymous",
				"Accept":             "application/json",
			})
		}

		if err == nil {
			err = v.DoJSON(req, &oauth)
		}
	}

	if err == nil {
		uri := fmt.Sprintf("%s/oauth2/%s/authorize", nissanAuthBaseURL, strings.Trim(oauth.Realm, "/"))

		data := url.Values{
			"client_id":     []string{nissanClientID},
			"redirect_uri":  []string{nissanRedirectURI},
			"response_type": []string{"code"},
			"scope":         []string{nissanScope},
			"nonce":         []string{"sdfdsfez"},
		}

		uri += "?" + data.Encode()
		req, err = request.New(http.MethodGet, uri, nil, map[string]string{
			"Cookie": "i18next=en-UK; amlbcookie=05; kauthSession=" + oauth.TokenID,
		})

		if err == nil {
			v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
			resp, err = v.Do(req)
			v.Client.CheckRedirect = nil

			if err == nil {
				resp.Body.Close()

				var location *url.URL
				if location, err = url.Parse(resp.Header.Get("Location")); err == nil {
					if code = location.Query().Get("code"); code == "" {
						err = fmt.Errorf("missing auth code: %v", location)
					}
				}
			}
		}
	}

	if err == nil {
		uri = fmt.Sprintf("%s/oauth2/%s/access_token", nissanAuthBaseURL, strings.Trim(oauth.Realm, "/"))

		data := url.Values{
			"code":          []string{code},
			"client_id":     []string{nissanClientID},
			"client_secret": []string{nissanClientSecret},
			"redirect_uri":  []string{nissanRedirectURI},
			"grant_type":    []string{"authorization_code"},
		}

		uri += "?" + data.Encode()
		req, err = request.New(http.MethodPost, uri, nil, request.URLEncoding)
		if err == nil {
			if err = v.DoJSON(req, &v.tokens); err == nil && v.tokens.AccessToken == "" {
				err = errors.New("missing access token")
			}
		}
	}

	if err == nil {
		uri = fmt.Sprintf("%s/v1/users/current", nissanUserAdapterBaseURL)

		var user struct{ UserID string }
		if req, err = request.New(http.MethodGet, uri, nil, nil); err == nil {
			if err = v.request(req, &user); err == nil {
				v.userID = user.UserID
			}
		}

		if v.userID == "" {
			err = errors.New("missing user id")
		}
	}

	return err
}

func (v *Nissan) refreshToken() error {
	uri := fmt.Sprintf("%s/oauth2/%s/access_token", nissanAuthBaseURL, nissanRealm)

	data := url.Values{
		"client_id":     []string{nissanClientID},
		"client_secret": []string{nissanClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {v.tokens.RefreshToken},
	}

	uri += "?" + data.Encode()
	req, err := request.New(http.MethodPost, uri, nil, request.URLEncoding)
	if err == nil {
		var tokens oidc.Tokens
		if err = v.DoJSON(req, &tokens); err == nil && v.tokens.AccessToken == "" {
			err = errors.New("missing access token")
		}
	}

	return err
}

// request executes given request and handles token refresh
func (v *Nissan) request(req *http.Request, res interface{}) error {
	req.Header.Set("Authorization", "Bearer "+v.tokens.AccessToken)
	err := v.DoJSON(req, &res)

	// repeat auth if error
	if err != nil {
		if err = v.refreshToken(); err != nil {
			err = v.authFlow()
		}
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+v.tokens.AccessToken)
			err = v.DoJSON(req, &res)
		}
	}

	return err
}

type nissanVehicles struct {
	Data []nissanVehicle
}

type nissanVehicle struct {
	VIN        string
	ModelName  string
	PictureURL string
}

func (v *Nissan) vehicles(userID string) ([]string, error) {
	uri := fmt.Sprintf("%s/v2/users/%s/cars", nissanUserBaseURL, userID)

	var res nissanVehicles
	req, err := request.New(http.MethodGet, uri, nil, nil)
	if err == nil {
		err = v.request(req, &res)
	}

	var vehicles []string
	if err == nil {
		for _, v := range res.Data {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// batteryAPI provides battery api response
func (v *Nissan) batteryAPI() (interface{}, error) {
	// refresh battery status
	uri := fmt.Sprintf("%s/v1/cars/%s/actions/refresh-battery-status", nissanCarAdapterBaseURL, v.vin)

	data := strings.NewReader(`{"data": {"type": "RefreshBatteryStatus"}}`)
	req, err := request.New(http.MethodPost, uri, data, map[string]string{
		"Content-Type": "application/vnd.api+json",
	})

	var res kamereon.Response
	if err == nil {
		err = v.request(req, &res)
	}

	// request battery status
	if err == nil {
		uri = fmt.Sprintf("%s/v1/cars/%s/battery-status", nissanCarAdapterBaseURL, v.vin)

		if req, err = request.New(http.MethodGet, uri, nil, nil); err == nil {
			err = v.request(req, &res)
		}
	}

	return res, err
}
